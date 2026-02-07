package domain

import (
	"testing"
	"time"
)

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityP0, "P0"},
		{PriorityP1, "P1"},
		{PriorityP2, "P2"},
		{PriorityP3, "P3"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected Priority
		hasError bool
	}{
		{"P0", PriorityP0, false},
		{"P1", PriorityP1, false},
		{"P2", PriorityP2, false},
		{"P3", PriorityP3, false},
		{"invalid", PriorityP3, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParsePriority(tt.input)
			if tt.hasError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestStateArchive(t *testing.T) {
	now := time.Now()
	state := &State{
		ID:          "STA-TASK-001",
		ProjectID:   "test-project",
		Type:        StateTypeTask,
		Status:      StatusInProgress,
		Title:       "Test Task",
		Description: "Test description",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Minute),
	}

	if !state.IsActive() {
		t.Error("expected state to be active")
	}

	archiveTime := time.Now()
	state.Archive("Completed successfully", archiveTime)

	if state.IsActive() {
		t.Error("expected state to be archived")
	}

	if state.Status != StatusArchived {
		t.Errorf("expected status archived, got %s", state.Status)
	}

	if state.Resolution != "Completed successfully" {
		t.Errorf("unexpected resolution: %s", state.Resolution)
	}

	if state.ArchivedAt == nil {
		t.Error("expected archived_at to be set")
	}

	if !state.ArchivedAt.Equal(archiveTime) {
		t.Errorf("expected archived_at %v, got %v", archiveTime, *state.ArchivedAt)
	}
}

func TestValidStockCategories(t *testing.T) {
	categories := ValidStockCategories()
	if len(categories) != 6 {
		t.Errorf("expected 6 categories, got %d", len(categories))
	}

	expected := map[StockCategory]bool{
		CategoryDesign:       true,
		CategoryRules:        true,
		CategoryManagement:   true,
		CategoryArchitecture: true,
		CategoryRequirement:  true,
		CategoryTest:         true,
	}

	for _, c := range categories {
		if !expected[c] {
			t.Errorf("unexpected category: %s", c)
		}
	}
}
