package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const tuiW = 72

var tuiIn = bufio.NewReader(os.Stdin)

// ─── Box helpers ──────────────────────────────────────────────────────────────

func tuiTop() string { return "╔" + strings.Repeat("═", tuiW-2) + "╗" }
func tuiBot() string { return "╚" + strings.Repeat("═", tuiW-2) + "╝" }
func tuiDiv() string { return "╠" + strings.Repeat("═", tuiW-2) + "╣" }

func tuiRow(s string) string {
	inner := tuiW - 4
	runes := []rune(s)
	if len(runes) > inner {
		s = string(runes[:inner-3]) + "..."
		runes = []rune(s)
	}
	pad := inner - len(runes)
	return "║ " + s + strings.Repeat(" ", pad) + " ║"
}

func tuiBlank() string { return tuiRow("") }

func tuiRead() string {
	s, _ := tuiIn.ReadString('\n')
	return strings.TrimSpace(s)
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// ─── Entry point ──────────────────────────────────────────────────────────────

func RunTUI() {
	storage, err := NewStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for showMainMenu(storage) {
	}
	clearScreen()
	fmt.Println("Goodbye!")
}

// ─── Main menu ────────────────────────────────────────────────────────────────

func showMainMenu(storage *Storage) bool {
	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow(fmt.Sprintf("  RickySpanish v%s  ─  Project Manager", version)))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [1]  All Projects"))
	fmt.Println(tuiRow("  [2]  Active"))
	fmt.Println(tuiRow("  [3]  On Hold"))
	fmt.Println(tuiRow("  [4]  Completed"))
	fmt.Println(tuiRow("  [5]  Archived"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [6]  Add New Project"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [0]  Exit"))
	fmt.Println(tuiBot())
	fmt.Print("  Choice: ")

	switch tuiRead() {
	case "1":
		showProjectList(storage, "")
	case "2":
		showProjectList(storage, string(StatusActive))
	case "3":
		showProjectList(storage, string(StatusOnHold))
	case "4":
		showProjectList(storage, string(StatusCompleted))
	case "5":
		showProjectList(storage, string(StatusArchived))
	case "6":
		showAddProject(storage)
	case "0", "q", "exit":
		return false
	}
	return true
}

// ─── Project list ─────────────────────────────────────────────────────────────

func showProjectList(storage *Storage, statusFilter string) {
	projects, err := storage.ListProjects()
	if err != nil {
		tuiError(err.Error())
		return
	}
	if statusFilter != "" {
		var filtered []Project
		for _, p := range projects {
			if string(p.Status) == statusFilter {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	clearScreen()
	title := "All Projects"
	if statusFilter != "" {
		title = tuiStatusLabel(statusFilter) + " Projects"
	}

	fmt.Println(tuiTop())
	fmt.Println(tuiRow(fmt.Sprintf("  %s (%d)", title, len(projects))))
	fmt.Println(tuiDiv())

	if len(projects) == 0 {
		fmt.Println(tuiRow("  No projects found."))
	} else {
		fmt.Println(tuiRow(fmt.Sprintf("  %-4s %-28s %-8s %-12s %s", "#", "NAME", "PRIORITY", "STATUS", "GOAL")))
		fmt.Println(tuiDiv())
		for i, p := range projects {
			goal := " "
			if p.CompanyGoal {
				goal = "Y"
			}
			line := fmt.Sprintf("  %-4s %-28s %-8s %-12s %s",
				fmt.Sprintf("[%d]", i+1), truncate(p.Name, 28), p.Priority, p.Status, goal)
			fmt.Println(tuiRow(line))
		}
	}

	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [0]  Back"))
	fmt.Println(tuiBot())
	fmt.Print("  Choice: ")

	choice := tuiRead()
	if choice == "0" || choice == "" {
		return
	}
	var idx int
	fmt.Sscanf(choice, "%d", &idx)
	if idx >= 1 && idx <= len(projects) {
		showProjectDetail(storage, projects[idx-1].ID)
	}
}

// ─── Project detail / edit ────────────────────────────────────────────────────

func showProjectDetail(storage *Storage, projectID string) {
	p, err := storage.GetProject(projectID)
	if err != nil {
		tuiError(err.Error())
		return
	}
	edited := *p

	for {
		clearScreen()
		goal := "No"
		if edited.CompanyGoal {
			goal = "Yes"
		}

		fmt.Println(tuiTop())
		fmt.Println(tuiRow(fmt.Sprintf("  PROJECT: %s", edited.Name)))
		fmt.Println(tuiDiv())
		fmt.Println(tuiRow(fmt.Sprintf("  ID:      %s", edited.ID)))
		fmt.Println(tuiRow(fmt.Sprintf("  Name:    %s", edited.Name)))
		fmt.Println(tuiRow(fmt.Sprintf("  Created: %s", edited.CreatedAt.Format("2006-01-02 15:04:05"))))
		fmt.Println(tuiRow(fmt.Sprintf("  Updated: %s", edited.UpdatedAt.Format("2006-01-02 15:04:05"))))
		fmt.Println(tuiDiv())
		fmt.Println(tuiRow(fmt.Sprintf("  [1]  Priority:     %s", edited.Priority)))
		fmt.Println(tuiRow(fmt.Sprintf("  [2]  Status:       %s", edited.Status)))
		fmt.Println(tuiRow(fmt.Sprintf("  [3]  Company Goal: %s", goal)))
		fmt.Println(tuiRow(fmt.Sprintf("  [4]  Directory:    %s", edited.Directory)))
		fmt.Println(tuiDiv())

		if len(edited.Notes) == 0 {
			fmt.Println(tuiRow("  Notes: (none)"))
		} else {
			fmt.Println(tuiRow(fmt.Sprintf("  Notes (%d):", len(edited.Notes))))
			for i, n := range edited.Notes {
				line := fmt.Sprintf("  [%d]  [%s]  %s  %s",
					i+1, n.ID[:8], n.CreatedAt.Format("2006-01-02"), truncate(n.Content, 38))
				fmt.Println(tuiRow(line))
			}
		}

		fmt.Println(tuiDiv())
		fmt.Println(tuiRow("  [N] Add Note   [R] Remove Note"))
		fmt.Println(tuiDiv())
		fmt.Println(tuiRow("  [S] Save   [C] Cancel   [D] Delete Project"))
		fmt.Println(tuiBot())
		fmt.Print("  Choice: ")

		switch strings.ToUpper(tuiRead()) {
		case "1":
			edited.Priority = tuiPickPriority(edited.Priority)
		case "2":
			edited.Status = tuiPickStatus(edited.Status)
		case "3":
			edited.CompanyGoal = !edited.CompanyGoal
		case "4":
			fmt.Print("  New directory: ")
			edited.Directory = tuiRead()
		case "N":
			fmt.Print("  Note text: ")
			content := tuiRead()
			if content != "" {
				edited.Notes = append(edited.Notes, Note{
					ID:        newID(),
					Content:   content,
					CreatedAt: time.Now().UTC(),
				})
			}
		case "R":
			if len(edited.Notes) == 0 {
				break
			}
			fmt.Print("  Remove note #: ")
			var noteIdx int
			fmt.Sscanf(tuiRead(), "%d", &noteIdx)
			if noteIdx >= 1 && noteIdx <= len(edited.Notes) {
				edited.Notes = append(edited.Notes[:noteIdx-1], edited.Notes[noteIdx:]...)
			}
		case "S":
			edited.UpdatedAt = time.Now().UTC()
			if err := storage.UpdateProject(edited); err != nil {
				tuiError(err.Error())
			}
			return
		case "C":
			return
		case "D":
			clearScreen()
			fmt.Println(tuiTop())
			fmt.Println(tuiRow(fmt.Sprintf("  Delete \"%s\"?", edited.Name)))
			fmt.Println(tuiDiv())
			fmt.Println(tuiRow("  This cannot be undone."))
			fmt.Println(tuiDiv())
			fmt.Println(tuiRow("  [Y] Yes, delete   [N] No, cancel"))
			fmt.Println(tuiBot())
			fmt.Print("  Choice: ")
			if strings.ToUpper(tuiRead()) == "Y" {
				if err := storage.DeleteProject(edited.ID); err != nil {
					tuiError(err.Error())
				}
			}
			return
		}
	}
}

// ─── Add project ──────────────────────────────────────────────────────────────

func showAddProject(storage *Storage) {
	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow("  Add New Project"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  Leave name blank for an auto-generated name."))
	fmt.Println(tuiBot())
	fmt.Println()

	fmt.Print("  Name (blank = auto-generate): ")
	name := tuiRead()

	priority := tuiPickPriority(PriorityMedium)
	status := tuiPickStatus(StatusActive)

	fmt.Print("  Company Goal? [y/N]: ")
	companyGoal := strings.ToLower(tuiRead()) == "y"

	fmt.Print("  Directory (optional): ")
	directory := tuiRead()

	p, err := buildProject(name, string(priority), companyGoal, string(status), directory)
	if err != nil {
		tuiError(err.Error())
		return
	}
	if err := storage.AddProject(p); err != nil {
		tuiError(err.Error())
		return
	}

	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow("  Project Created"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow(fmt.Sprintf("  Name:     %s", p.Name)))
	fmt.Println(tuiRow(fmt.Sprintf("  ID:       %s", p.ID)))
	fmt.Println(tuiRow(fmt.Sprintf("  Priority: %s", p.Priority)))
	fmt.Println(tuiRow(fmt.Sprintf("  Status:   %s", p.Status)))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  Press Enter to return to main menu."))
	fmt.Println(tuiBot())
	tuiRead()
}

// ─── Pickers ──────────────────────────────────────────────────────────────────

func tuiPickPriority(current Priority) Priority {
	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow(fmt.Sprintf("  Select Priority  (current: %s)", current)))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [1]  Low"))
	fmt.Println(tuiRow("  [2]  Medium"))
	fmt.Println(tuiRow("  [3]  High"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [0]  Keep current"))
	fmt.Println(tuiBot())
	fmt.Print("  Choice: ")
	switch tuiRead() {
	case "1":
		return PriorityLow
	case "2":
		return PriorityMedium
	case "3":
		return PriorityHigh
	}
	return current
}

func tuiPickStatus(current Status) Status {
	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow(fmt.Sprintf("  Select Status  (current: %s)", current)))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [1]  Active"))
	fmt.Println(tuiRow("  [2]  On Hold"))
	fmt.Println(tuiRow("  [3]  Completed"))
	fmt.Println(tuiRow("  [4]  Archived"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  [0]  Keep current"))
	fmt.Println(tuiBot())
	fmt.Print("  Choice: ")
	switch tuiRead() {
	case "1":
		return StatusActive
	case "2":
		return StatusOnHold
	case "3":
		return StatusCompleted
	case "4":
		return StatusArchived
	}
	return current
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func tuiError(msg string) {
	clearScreen()
	fmt.Println(tuiTop())
	fmt.Println(tuiRow("  Error"))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  " + msg))
	fmt.Println(tuiDiv())
	fmt.Println(tuiRow("  Press Enter to continue."))
	fmt.Println(tuiBot())
	tuiRead()
}

func tuiStatusLabel(s string) string {
	switch s {
	case "active":
		return "Active"
	case "on_hold":
		return "On Hold"
	case "completed":
		return "Completed"
	case "archived":
		return "Archived"
	}
	return s
}
