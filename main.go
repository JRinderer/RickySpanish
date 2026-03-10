package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const version = "1.0.1"

func main() {
	if len(os.Args) < 2 {
		RunTUI()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "add":
		cmdAdd(args)
	case "list", "ls":
		cmdList(args)
	case "get", "show":
		cmdGet(args)
	case "update":
		cmdUpdate(args)
	case "delete", "rm":
		cmdDelete(args)
	case "note":
		cmdNote(args)
	case "notes":
		cmdNotes(args)
	case "delete-note":
		cmdDeleteNote(args)
	case "task-add":
		cmdTaskAdd(args)
	case "task-list":
		cmdTaskList(args)
	case "task-update":
		cmdTaskUpdate(args)
	case "task-comment":
		cmdTaskComment(args)
	case "task-delete":
		cmdTaskDelete(args)
	case "ui":
		RunTUI()
	case "serve":
		cmdServe()
	case "version":
		fmt.Printf("rickspanish %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	name := fs.String("name", "", "Project name (auto-generated if omitted)")
	priority := fs.String("priority", "medium", "Priority: low, medium, high")
	companyGoal := fs.Bool("company-goal", false, "Mark as related to a company goal")
	status := fs.String("status", "active", "Status: active, on_hold, completed, archived")
	directory := fs.String("dir", "", "Path to project directory")
	fs.Parse(args)

	storage, err := NewStorage()
	dieOnErr(err)

	p, err := buildProject(*name, *priority, *companyGoal, *status, *directory)
	dieOnErr(err)

	dieOnErr(storage.AddProject(p))
	fmt.Printf("Project created successfully.\n\n%s", p.String())
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	status := fs.String("status", "", "Filter by status")
	priority := fs.String("priority", "", "Filter by priority")
	companyGoalStr := fs.String("company-goal", "", "Filter by company goal: true or false")
	fs.Parse(args)

	storage, err := NewStorage()
	dieOnErr(err)

	projects, err := storage.ListProjects()
	dieOnErr(err)

	var companyGoalFilter *bool
	if *companyGoalStr != "" {
		b := *companyGoalStr == "true" || *companyGoalStr == "1" || *companyGoalStr == "yes"
		companyGoalFilter = &b
	}

	projects = filterProjects(projects, *status, *priority, companyGoalFilter)

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	fmt.Printf("%-36s  %-30s  %-8s  %-12s  %-4s\n",
		"ID", "NAME", "PRIORITY", "STATUS", "GOAL")
	fmt.Println(strings.Repeat("-", 100))
	for _, p := range projects {
		goal := " "
		if p.CompanyGoal {
			goal = "✓"
		}
		fmt.Printf("%-36s  %-30s  %-8s  %-12s  %-4s\n",
			p.ID, truncate(p.Name, 30), p.Priority, p.Status, goal)
	}
}

func cmdGet(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish get <id-or-name>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	p, err := storage.GetProject(args[0])
	dieOnErr(err)
	fmt.Print(p.String())
}

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	name := fs.String("name", "", "New name")
	priority := fs.String("priority", "", "New priority: low, medium, high")
	companyGoalStr := fs.String("company-goal", "", "Set company goal: true or false")
	status := fs.String("status", "", "New status: active, on_hold, completed, archived")
	directory := fs.String("dir", "", "New project directory")
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish update <id> [flags]")
		os.Exit(1)
	}
	id := remaining[0]

	storage, err := NewStorage()
	dieOnErr(err)

	p, err := storage.GetProject(id)
	dieOnErr(err)

	var namePtr, priorityPtr, statusPtr, dirPtr *string
	if *name != "" {
		namePtr = name
	}
	if *priority != "" {
		priorityPtr = priority
	}
	if *status != "" {
		statusPtr = status
	}
	if *directory != "" {
		dirPtr = directory
	}
	var companyGoalPtr *bool
	if *companyGoalStr != "" {
		b := *companyGoalStr == "true" || *companyGoalStr == "1" || *companyGoalStr == "yes"
		companyGoalPtr = &b
	}

	applyUpdates(p, namePtr, priorityPtr, companyGoalPtr, statusPtr, dirPtr)
	dieOnErr(storage.UpdateProject(*p))
	fmt.Printf("Project updated:\n\n%s", p.String())
}

func cmdDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish delete <id-or-name>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	dieOnErr(storage.DeleteProject(args[0]))
	fmt.Printf("Project %q deleted.\n", args[0])
}

func cmdNote(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish note <id-or-name> <note text>")
		os.Exit(1)
	}
	projectID := args[0]
	content := strings.Join(args[1:], " ")

	storage, err := NewStorage()
	dieOnErr(err)

	noteID := newID()
	dieOnErr(storage.AddNote(projectID, noteID, content))
	fmt.Printf("Note added (ID: %s).\n", noteID[:8])
}

func cmdNotes(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish notes <id-or-name>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	p, err := storage.GetProject(args[0])
	dieOnErr(err)

	if len(p.Notes) == 0 {
		fmt.Println("No notes for this project.")
		return
	}
	for _, n := range p.Notes {
		fmt.Printf("[%s] %s\n%s\n\n",
			n.ID[:8],
			n.CreatedAt.Format("2006-01-02 15:04:05"),
			n.Content)
	}
}

func cmdDeleteNote(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish delete-note <project-id> <note-id>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	dieOnErr(storage.DeleteNote(args[0], args[1]))
	fmt.Printf("Note %s deleted.\n", args[1])
}

func cmdTaskAdd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish task-add <project-id> <description>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	t := now()
	task := Task{
		ID:              newID(),
		Description:     strings.Join(args[1:], " "),
		Comments:        []string{},
		Status:          StatusActive,
		StatusChangedAt: t,
		CreatedAt:       t,
	}
	dieOnErr(storage.AddTask(args[0], task))
	fmt.Printf("Task added (ID: %s).\n", task.ID[:8])
}

func cmdTaskList(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish task-list <project-id>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	p, err := storage.GetProject(args[0])
	dieOnErr(err)

	if len(p.Tasks) == 0 {
		fmt.Println("No tasks for this project.")
		return
	}
	fmt.Printf("%-8s  %-40s  %-12s  %s\n", "ID", "DESCRIPTION", "STATUS", "CHANGED")
	fmt.Println(strings.Repeat("-", 80))
	for _, t := range p.Tasks {
		fmt.Printf("%-8s  %-40s  %-12s  %s\n",
			t.ID[:8], truncate(t.Description, 40), t.Status,
			t.StatusChangedAt.Format("2006-01-02"))
	}
}

func cmdTaskUpdate(args []string) {
	fs := flag.NewFlagSet("task-update", flag.ExitOnError)
	status := fs.String("status", "", "New status: active, on_hold, completed, archived")
	description := fs.String("description", "", "New description")
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish task-update <project-id> <task-id> [--status ...] [--description ...]")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	p, err := storage.GetProject(remaining[0])
	dieOnErr(err)

	taskID := remaining[1]
	for _, t := range p.Tasks {
		if t.ID == taskID || (len(t.ID) >= 8 && t.ID[:8] == taskID) {
			if *description != "" {
				t.Description = *description
			}
			if *status != "" {
				t.Status = Status(*status)
			}
			dieOnErr(storage.UpdateTask(p.ID, t))
			fmt.Printf("Task %s updated.\n", t.ID[:8])
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Task %q not found.\n", taskID)
	os.Exit(1)
}

func cmdTaskComment(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish task-comment <project-id> <task-id> <comment text>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	comment := strings.Join(args[2:], " ")
	dieOnErr(storage.AddTaskComment(args[0], args[1], comment))
	fmt.Println("Comment added.")
}

func cmdTaskDelete(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: rickspanish task-delete <project-id> <task-id>")
		os.Exit(1)
	}
	storage, err := NewStorage()
	dieOnErr(err)

	dieOnErr(storage.DeleteTask(args[0], args[1]))
	fmt.Printf("Task %s deleted.\n", args[1])
}

func cmdServe() {
	storage, err := NewStorage()
	dieOnErr(err)

	fmt.Fprintln(os.Stderr, "RickySpanish MCP server starting on stdio...")
	server := newMCPServer(storage)
	if err := server.run(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func now() time.Time {
	return time.Now().UTC()
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func dieOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`RickySpanish - Project Management CLI  v` + version + `

USAGE:
  rickspanish <command> [flags]

COMMANDS:
  add           Create a new project
  list (ls)     List projects
  get (show)    Show project details
  update        Update a project
  delete (rm)   Delete a project
  note          Add a note to a project
  notes         List notes for a project
  delete-note   Delete a note from a project
  task-add      Add a task to a project
  task-list     List tasks for a project
  task-update   Update a task's status or description
  task-comment  Add a comment to a task
  task-delete   Delete a task from a project
  ui            Launch interactive menu interface
  serve         Start MCP server (for use with Claude)
  version       Show version

ADD FLAGS:
  --name         Project name (auto-generated from military ops if omitted)
  --priority     low | medium | high  (default: medium)
  --company-goal Mark as related to a company goal
  --status       active | on_hold | completed | archived  (default: active)
  --dir          Path to project directory

UPDATE FLAGS:
  rickspanish update <id> [same flags as add]

LIST FLAGS:
  --status       Filter by status
  --priority     Filter by priority
  --company-goal Filter: true | false

EXAMPLES:
  rickspanish add --name "Website Redesign" --priority high --company-goal
  rickspanish add                                      # auto-generated name
  rickspanish list --status active --priority high
  rickspanish get <id-or-name>
  rickspanish update <id> --status completed
  rickspanish note <id> "Meeting with stakeholders today"
  rickspanish serve                                    # start MCP server

`)
}
