package proactive

import (
	"context"
	"fmt"
	"strings"

	"bruce/internal/ai"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/scheduler"
)

// ListTool returns all active and paused proactive tasks.
type ListTool struct {
	repo repository.ProactiveTaskRepository
}

// NewListTool returns a new ListTool.
func NewListTool(repo repository.ProactiveTaskRepository) *ListTool {
	return &ListTool{repo: repo}
}

func (t *ListTool) Name() string {
	return "proactive_list"
}

func (t *ListTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "proactive_list",
		Description: "List all scheduled reports (crons) and background condition watches with their status and next run times.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional session ID filter. If omitted or empty, lists all tasks",
				},
			},
		},
	}
}

func (t *ListTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	sessionID, _ := input["session_id"].(string)

	var tasks []domain.ProactiveTask
	var err error

	if strings.TrimSpace(sessionID) != "" {
		tasks, err = t.repo.ListBySession(ctx, sessionID)
	} else {
		tasks, err = t.repo.ListAll(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("list proactive tasks: %w", err)
	}

	if len(tasks) == 0 {
		return "No proactive tasks or scheduled reports found.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d proactive task(s):\n\n", len(tasks)))

	for i, task := range tasks {
		status := "🟢 Active"
		if !task.IsActive {
			status = "⏸️ Paused"
		}
		loc := scheduler.ResolveTimezone(task.Timezone, "")
		nextRunStr := task.NextRunAt.In(loc).Format("2006-01-02 15:04:05 MST")
		if !task.IsActive {
			nextRunStr = "(paused)"
		}

		sb.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, task.Title, status))
		sb.WriteString(fmt.Sprintf("   - ID: `%s`\n", task.ID))
		sb.WriteString(fmt.Sprintf("   - Type: %s | Schedule: `%s` (%s)\n", task.TaskType, task.ScheduleExpr, task.Timezone))
		sb.WriteString(fmt.Sprintf("   - Destination: %s (%s)\n", task.TargetConnector, task.TargetChannelID))
		sb.WriteString(fmt.Sprintf("   - Next Run: %s\n", nextRunStr))
		sb.WriteString(fmt.Sprintf("   - Prompt/Condition: %q\n\n", task.PromptCondition))
	}

	return strings.TrimSpace(sb.String()), nil
}
