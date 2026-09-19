package proactive

import (
	"context"
	"fmt"
	"strings"

	"bruce/internal/ai"
	"bruce/internal/repository"
)

// DeleteTool cancels and permanently deletes a proactive task.
type DeleteTool struct {
	repo repository.ProactiveTaskRepository
}

// NewDeleteTool returns a new DeleteTool.
func NewDeleteTool(repo repository.ProactiveTaskRepository) *DeleteTool {
	return &DeleteTool{repo: repo}
}

func (t *DeleteTool) Name() string {
	return "proactive_delete"
}

func (t *DeleteTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "proactive_delete",
		Description: "Permanently delete/cancel a scheduled report or ambient watch by task ID or title.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id_or_title": map[string]interface{}{
					"type":        "string",
					"description": "ID or exact title of the task to delete",
				},
			},
			"required": []string{"task_id_or_title"},
		},
	}
}

func (t *DeleteTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	target, _ := input["task_id_or_title"].(string)
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("task_id_or_title is required")
	}

	task, err := findTask(ctx, t.repo, target)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("no task found matching %q", target)
	}

	if err := t.repo.Delete(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("delete task: %w", err)
	}

	return fmt.Sprintf("Proactive task '%s' (ID: %s) has been successfully deleted.", task.Title, task.ID), nil
}
