package proactive

import (
	"context"
	"fmt"
	"strings"

	"bruce/internal/ai"
	"bruce/internal/domain"
	"bruce/internal/repository"
)

// ToggleTool pauses or resumes a proactive task.
type ToggleTool struct {
	repo repository.ProactiveTaskRepository
}

// NewToggleTool returns a new ToggleTool.
func NewToggleTool(repo repository.ProactiveTaskRepository) *ToggleTool {
	return &ToggleTool{repo: repo}
}

func (t *ToggleTool) Name() string {
	return "proactive_toggle"
}

func (t *ToggleTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "proactive_toggle",
		Description: "Pause or resume a scheduled report or ambient watch by task ID or title.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id_or_title": map[string]interface{}{
					"type":        "string",
					"description": "ID or exact title of the task to toggle",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"pause", "resume"},
					"description": "Whether to pause or resume the task",
				},
			},
			"required": []string{"task_id_or_title", "action"},
		},
	}
}

func (t *ToggleTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	target, _ := input["task_id_or_title"].(string)
	action, _ := input["action"].(string)

	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("task_id_or_title is required")
	}

	action = strings.ToLower(strings.TrimSpace(action))
	if action != "pause" && action != "resume" {
		return nil, fmt.Errorf("action must be 'pause' or 'resume'")
	}

	task, err := findTask(ctx, t.repo, target)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("no task found matching %q", target)
	}

	isActive := (action == "resume")
	if err := t.repo.UpdateStatus(ctx, task.ID, isActive); err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}

	verb := "paused"
	if isActive {
		verb = "resumed"
	}

	return fmt.Sprintf("Task '%s' (ID: %s) has been successfully %s.", task.Title, task.ID, verb), nil
}

func findTask(ctx context.Context, repo repository.ProactiveTaskRepository, target string) (*domain.ProactiveTask, error) {
	// Try finding by direct ID first
	t, err := repo.GetByID(ctx, target)
	if err == nil && t != nil {
		return t, nil
	}

	// Try matching by title
	all, err := repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	targetLower := strings.ToLower(target)
	for _, task := range all {
		if strings.ToLower(task.Title) == targetLower || task.ID == target {
			return &task, nil
		}
	}

	return nil, nil
}
