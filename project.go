package main

import (
	"fmt"
	"strings"
	"time"
)

type Priority string
type Status string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"

	StatusActive    Status = "active"
	StatusOnHold    Status = "on_hold"
	StatusCompleted Status = "completed"
	StatusArchived  Status = "archived"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusOnHold, StatusCompleted, StatusArchived:
		return true
	}
	return false
}

type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Priority    Priority  `json:"priority"`
	CompanyGoal bool      `json:"company_goal"`
	Status      Status    `json:"status"`
	Notes       []Note    `json:"notes"`
	Directory   string    `json:"directory"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Project) String() string {
	var sb strings.Builder
	companyGoal := "no"
	if p.CompanyGoal {
		companyGoal = "yes"
	}
	fmt.Fprintf(&sb, "ID:           %s\n", p.ID)
	fmt.Fprintf(&sb, "Name:         %s\n", p.Name)
	fmt.Fprintf(&sb, "Priority:     %s\n", p.Priority)
	fmt.Fprintf(&sb, "Company Goal: %s\n", companyGoal)
	fmt.Fprintf(&sb, "Status:       %s\n", p.Status)
	fmt.Fprintf(&sb, "Directory:    %s\n", p.Directory)
	fmt.Fprintf(&sb, "Created:      %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "Updated:      %s\n", p.UpdatedAt.Format("2006-01-02 15:04:05"))
	if len(p.Notes) > 0 {
		fmt.Fprintf(&sb, "Notes (%d):\n", len(p.Notes))
		for _, n := range p.Notes {
			fmt.Fprintf(&sb, "  [%s] %s: %s\n",
				n.ID[:8],
				n.CreatedAt.Format("2006-01-02"),
				n.Content)
		}
	}
	return sb.String()
}

type Database struct {
	Projects []Project `json:"projects"`
}
