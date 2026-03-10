package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ─── JSON-RPC 2.0 Types ───────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string   `json:"jsonrpc"`
	ID      any      `json:"id"`
	Result  any      `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ─── MCP Protocol Types ───────────────────────────────────────────────────────

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpCapabilities struct {
	Tools *struct{} `json:"tools,omitempty"`
}

type mcpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	ServerInfo      mcpServerInfo   `json:"serverInfo"`
	Capabilities    mcpCapabilities `json:"capabilities"`
}

type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// ─── MCP Server ──────────────────────────────────────────────────────────────

type mcpServer struct {
	storage *Storage
	in      *bufio.Reader
	out     io.Writer
}

func newMCPServer(storage *Storage) *mcpServer {
	return &mcpServer{
		storage: storage,
		in:      bufio.NewReader(os.Stdin),
		out:     os.Stdout,
	}
}

func (s *mcpServer) run() error {
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(req)
	}
}

func (s *mcpServer) handleRequest(req rpcRequest) {
	switch req.Method {
	case "initialize":
		caps := mcpCapabilities{Tools: &struct{}{}}
		s.sendResult(req.ID, mcpInitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      mcpServerInfo{Name: "rickspanish", Version: "1.0.0"},
			Capabilities:    caps,
		})

	case "initialized":
		// Notification — no response needed

	case "tools/list":
		s.sendResult(req.ID, mcpToolsListResult{Tools: s.toolList()})

	case "tools/call":
		s.handleToolCall(req)

	case "ping":
		s.sendResult(req.ID, map[string]any{})

	default:
		if req.ID != nil {
			s.sendError(req.ID, -32601, "Method not found: "+req.Method)
		}
	}
}

func (s *mcpServer) handleToolCall(req rpcRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	var (
		result string
		callErr error
	)

	switch params.Name {
	case "add_project":
		result, callErr = s.toolAddProject(params.Arguments)
	case "list_projects":
		result, callErr = s.toolListProjects(params.Arguments)
	case "get_project":
		result, callErr = s.toolGetProject(params.Arguments)
	case "update_project":
		result, callErr = s.toolUpdateProject(params.Arguments)
	case "delete_project":
		result, callErr = s.toolDeleteProject(params.Arguments)
	case "add_note":
		result, callErr = s.toolAddNote(params.Arguments)
	case "delete_note":
		result, callErr = s.toolDeleteNote(params.Arguments)
	case "add_task":
		result, callErr = s.toolAddTask(params.Arguments)
	case "list_tasks":
		result, callErr = s.toolListTasks(params.Arguments)
	case "update_task":
		result, callErr = s.toolUpdateTask(params.Arguments)
	case "delete_task":
		result, callErr = s.toolDeleteTask(params.Arguments)
	case "add_task_comment":
		result, callErr = s.toolAddTaskComment(params.Arguments)
	default:
		s.sendError(req.ID, -32601, "Unknown tool: "+params.Name)
		return
	}

	if callErr != nil {
		s.sendResult(req.ID, mcpCallResult{
			Content: []mcpContent{{Type: "text", Text: "Error: " + callErr.Error()}},
			IsError: true,
		})
		return
	}
	s.sendResult(req.ID, mcpCallResult{
		Content: []mcpContent{{Type: "text", Text: result}},
	})
}

// ─── Tool Implementations ─────────────────────────────────────────────────────

func (s *mcpServer) toolAddProject(raw json.RawMessage) (string, error) {
	var args struct {
		Name        string `json:"name"`
		Priority    string `json:"priority"`
		CompanyGoal bool   `json:"company_goal"`
		Status      string `json:"status"`
		Directory   string `json:"directory"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	p, err := buildProject(args.Name, args.Priority, args.CompanyGoal, args.Status, args.Directory)
	if err != nil {
		return "", err
	}
	if err := s.storage.AddProject(p); err != nil {
		return "", err
	}
	return fmt.Sprintf("Project created:\n%s", p.String()), nil
}

func (s *mcpServer) toolListProjects(raw json.RawMessage) (string, error) {
	var args struct {
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		CompanyGoal *bool  `json:"company_goal"`
	}
	_ = json.Unmarshal(raw, &args)

	projects, err := s.storage.ListProjects()
	if err != nil {
		return "", err
	}

	projects = filterProjects(projects, args.Status, args.Priority, args.CompanyGoal)
	if len(projects) == 0 {
		return "No projects found.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d project(s):\n\n", len(projects))
	for _, p := range projects {
		fmt.Fprintf(&sb, "%s\n---\n", p.String())
	}
	return sb.String(), nil
}

func (s *mcpServer) toolGetProject(raw json.RawMessage) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	p, err := s.storage.GetProject(args.ID)
	if err != nil {
		return "", err
	}
	return p.String(), nil
}

func (s *mcpServer) toolUpdateProject(raw json.RawMessage) (string, error) {
	var args struct {
		ID          string  `json:"id"`
		Name        *string `json:"name"`
		Priority    *string `json:"priority"`
		CompanyGoal *bool   `json:"company_goal"`
		Status      *string `json:"status"`
		Directory   *string `json:"directory"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	p, err := s.storage.GetProject(args.ID)
	if err != nil {
		return "", err
	}
	applyUpdates(p, args.Name, args.Priority, args.CompanyGoal, args.Status, args.Directory)
	if err := s.storage.UpdateProject(*p); err != nil {
		return "", err
	}
	return fmt.Sprintf("Project updated:\n%s", p.String()), nil
}

func (s *mcpServer) toolDeleteProject(raw json.RawMessage) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.storage.DeleteProject(args.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Project %q deleted.", args.ID), nil
}

func (s *mcpServer) toolAddNote(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID string `json:"project_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	noteID := newID()
	if err := s.storage.AddNote(args.ProjectID, noteID, args.Content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Note %s added to project %s.", noteID[:8], args.ProjectID), nil
}

func (s *mcpServer) toolDeleteNote(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID string `json:"project_id"`
		NoteID    string `json:"note_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.storage.DeleteNote(args.ProjectID, args.NoteID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Note %s deleted from project %s.", args.NoteID, args.ProjectID), nil
}

func (s *mcpServer) toolAddTask(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID   string `json:"project_id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Description == "" {
		return "", fmt.Errorf("description is required")
	}
	st := Status(args.Status)
	if args.Status == "" {
		st = StatusActive
	} else if !st.Valid() {
		return "", fmt.Errorf("invalid status %q", args.Status)
	}
	t := now()
	task := Task{
		ID:              newID(),
		Description:     args.Description,
		Comments:        []string{},
		Status:          st,
		StatusChangedAt: t,
		CreatedAt:       t,
	}
	if err := s.storage.AddTask(args.ProjectID, task); err != nil {
		return "", err
	}
	return fmt.Sprintf("Task %s added to project %s.", task.ID[:8], args.ProjectID), nil
}

func (s *mcpServer) toolListTasks(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	p, err := s.storage.GetProject(args.ProjectID)
	if err != nil {
		return "", err
	}
	if len(p.Tasks) == 0 {
		return "No tasks for this project.", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tasks for %q (%d):\n\n", p.Name, len(p.Tasks))
	for _, t := range p.Tasks {
		fmt.Fprintf(&sb, "[%s] %-12s %s\n", t.ID[:8], t.Status, t.Description)
		fmt.Fprintf(&sb, "  Changed: %s\n", t.StatusChangedAt.Format("2006-01-02 15:04:05"))
		for _, c := range t.Comments {
			fmt.Fprintf(&sb, "  Comment: %s\n", c)
		}
		fmt.Fprintln(&sb)
	}
	return sb.String(), nil
}

func (s *mcpServer) toolUpdateTask(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID   string  `json:"project_id"`
		TaskID      string  `json:"task_id"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	p, err := s.storage.GetProject(args.ProjectID)
	if err != nil {
		return "", err
	}
	for _, t := range p.Tasks {
		if t.ID == args.TaskID || (len(t.ID) >= 8 && t.ID[:8] == args.TaskID) {
			if args.Description != nil {
				t.Description = *args.Description
			}
			if args.Status != nil {
				t.Status = Status(*args.Status)
			}
			if err := s.storage.UpdateTask(p.ID, t); err != nil {
				return "", err
			}
			return fmt.Sprintf("Task %s updated.", t.ID[:8]), nil
		}
	}
	return "", fmt.Errorf("task %q not found", args.TaskID)
}

func (s *mcpServer) toolDeleteTask(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID string `json:"project_id"`
		TaskID    string `json:"task_id"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.storage.DeleteTask(args.ProjectID, args.TaskID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Task %s deleted.", args.TaskID), nil
}

func (s *mcpServer) toolAddTaskComment(raw json.RawMessage) (string, error) {
	var args struct {
		ProjectID string `json:"project_id"`
		TaskID    string `json:"task_id"`
		Comment   string `json:"comment"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if err := s.storage.AddTaskComment(args.ProjectID, args.TaskID, args.Comment); err != nil {
		return "", err
	}
	return fmt.Sprintf("Comment added to task %s.", args.TaskID), nil
}

// ─── Tool Schema Definitions ──────────────────────────────────────────────────

func (s *mcpServer) toolList() []mcpTool {
	return []mcpTool{
		{
			Name:        "add_project",
			Description: "Create a new project. Name is auto-generated from WWII/Korea military operations if omitted.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"name":         {"type": "string", "description": "Project name (auto-generated if omitted)"},
					"priority":     {"type": "string", "enum": ["low", "medium", "high"], "description": "Project priority"},
					"company_goal": {"type": "boolean", "description": "Whether this project relates to a company goal"},
					"status":       {"type": "string", "enum": ["active", "on_hold", "completed", "archived"], "description": "Project status"},
					"directory":    {"type": "string", "description": "Path to project directory for supporting documents"}
				}
			}`),
		},
		{
			Name:        "list_projects",
			Description: "List all projects, optionally filtered by status, priority, or company goal.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"status":       {"type": "string", "enum": ["active", "on_hold", "completed", "archived"]},
					"priority":     {"type": "string", "enum": ["low", "medium", "high"]},
					"company_goal": {"type": "boolean"}
				}
			}`),
		},
		{
			Name:        "get_project",
			Description: "Get full details of a project by ID or name.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["id"],
				"properties": {
					"id": {"type": "string", "description": "Project ID or name"}
				}
			}`),
		},
		{
			Name:        "update_project",
			Description: "Update one or more fields of an existing project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["id"],
				"properties": {
					"id":           {"type": "string", "description": "Project ID"},
					"name":         {"type": "string"},
					"priority":     {"type": "string", "enum": ["low", "medium", "high"]},
					"company_goal": {"type": "boolean"},
					"status":       {"type": "string", "enum": ["active", "on_hold", "completed", "archived"]},
					"directory":    {"type": "string"}
				}
			}`),
		},
		{
			Name:        "delete_project",
			Description: "Permanently delete a project by ID or name.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["id"],
				"properties": {
					"id": {"type": "string", "description": "Project ID or name"}
				}
			}`),
		},
		{
			Name:        "add_note",
			Description: "Add a note or comment to a project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "content"],
				"properties": {
					"project_id": {"type": "string", "description": "Project ID or name"},
					"content":    {"type": "string", "description": "Note content"}
				}
			}`),
		},
		{
			Name:        "delete_note",
			Description: "Delete a note from a project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "note_id"],
				"properties": {
					"project_id": {"type": "string"},
					"note_id":    {"type": "string", "description": "Full note ID or first 8 characters"}
				}
			}`),
		},
		{
			Name:        "add_task",
			Description: "Add a task to a project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "description"],
				"properties": {
					"project_id":  {"type": "string", "description": "Project ID or name"},
					"description": {"type": "string", "description": "Task description"},
					"status":      {"type": "string", "enum": ["active", "on_hold", "completed", "archived"], "description": "Initial status (default: active)"}
				}
			}`),
		},
		{
			Name:        "list_tasks",
			Description: "List all tasks for a project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id"],
				"properties": {
					"project_id": {"type": "string", "description": "Project ID or name"}
				}
			}`),
		},
		{
			Name:        "update_task",
			Description: "Update a task's description or status. Status changes are automatically timestamped.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "task_id"],
				"properties": {
					"project_id":  {"type": "string"},
					"task_id":     {"type": "string", "description": "Full task ID or first 8 characters"},
					"description": {"type": "string"},
					"status":      {"type": "string", "enum": ["active", "on_hold", "completed", "archived"]}
				}
			}`),
		},
		{
			Name:        "delete_task",
			Description: "Delete a task from a project.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "task_id"],
				"properties": {
					"project_id": {"type": "string"},
					"task_id":    {"type": "string", "description": "Full task ID or first 8 characters"}
				}
			}`),
		},
		{
			Name:        "add_task_comment",
			Description: "Add a comment to a task.",
			InputSchema: mustJSON(`{
				"type": "object",
				"required": ["project_id", "task_id", "comment"],
				"properties": {
					"project_id": {"type": "string"},
					"task_id":    {"type": "string", "description": "Full task ID or first 8 characters"},
					"comment":    {"type": "string"}
				}
			}`),
		},
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (s *mcpServer) sendResult(id any, result any) {
	s.send(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *mcpServer) sendError(id any, code int, msg string) {
	s.send(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *mcpServer) send(resp rpcResponse) {
	data, _ := json.Marshal(resp)
	fmt.Fprintf(s.out, "%s\n", data)
}

func mustJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}

// ─── Shared business logic helpers ───────────────────────────────────────────

func buildProject(name, priority string, companyGoal bool, status, directory string) (Project, error) {
	if name == "" {
		name = generateOperationName()
	}
	p := Priority(priority)
	if priority == "" {
		p = PriorityMedium
	} else if !p.Valid() {
		return Project{}, fmt.Errorf("invalid priority %q (use low, medium, high)", priority)
	}
	st := Status(status)
	if status == "" {
		st = StatusActive
	} else if !st.Valid() {
		return Project{}, fmt.Errorf("invalid status %q", status)
	}
	t := now()
	return Project{
		ID:          newID(),
		Name:        name,
		Priority:    p,
		CompanyGoal: companyGoal,
		Status:      st,
		Notes:       []Note{},
		Tasks:       []Task{},
		Directory:   directory,
		CreatedAt:   t,
		UpdatedAt:   t,
	}, nil
}

func applyUpdates(p *Project, name, priority *string, companyGoal *bool, status, directory *string) {
	if name != nil {
		p.Name = *name
	}
	if priority != nil {
		p.Priority = Priority(*priority)
	}
	if companyGoal != nil {
		p.CompanyGoal = *companyGoal
	}
	if status != nil {
		p.Status = Status(*status)
	}
	if directory != nil {
		p.Directory = *directory
	}
	p.UpdatedAt = time.Now().UTC()
}

func filterProjects(projects []Project, status, priority string, companyGoal *bool) []Project {
	var out []Project
	for _, p := range projects {
		if status != "" && string(p.Status) != status {
			continue
		}
		if priority != "" && string(p.Priority) != priority {
			continue
		}
		if companyGoal != nil && p.CompanyGoal != *companyGoal {
			continue
		}
		out = append(out, p)
	}
	return out
}
