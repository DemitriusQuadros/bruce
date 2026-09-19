package proactive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/scheduler"
)

// CreateTool allows Bruce to register a new recurring cron schedule or ambient watch.
type CreateTool struct {
	repo repository.ProactiveTaskRepository
	cfg  *config.Config
}

// NewCreateTool returns a new CreateTool.
func NewCreateTool(repo repository.ProactiveTaskRepository, cfg *config.Config) *CreateTool {
	return &CreateTool{repo: repo, cfg: cfg}
}

func (t *CreateTool) Name() string {
	return "proactive_create"
}

func (t *CreateTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "proactive_create",
		Description: "Create a scheduled recurring report (cron) or ambient monitoring watch. Runs in the background and sends proactive messages to messaging channels.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Short descriptive title for the task (e.g. 'Daily Standup Briefing', 'Contractor Email Watch')",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"cron", "watch"},
					"description": "'cron' for time-of-day scheduled reports; 'watch' for interval condition monitoring",
				},
				"schedule": map[string]interface{}{
					"type":        "string",
					"description": "For cron: 5-token cron expression (e.g. '0 9 * * 1-5'). For watch: interval in minutes as a number string (e.g. '30', minimum 5)",
				},
				"prompt_condition": map[string]interface{}{
					"type":        "string",
					"description": "For cron: instructions on what report to compile. For watch: natural language trigger condition (e.g. 'emails from contractors about invoices')",
				},
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "Optional IANA timezone (e.g. 'America/Sao_Paulo', 'America/New_York'). Defaults to local host machine timezone",
				},
				"target_connector": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"whatsapp", "discord", "telegram", "web"},
					"description": "Optional destination messaging platform to receive alerts/reports. Defaults to current channel",
				},
				"target_channel_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional destination channel ID / phone number. Defaults to current channel",
				},
				"target_tools": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Optional list of specific tool names for ambient watches (e.g. ['gmail_search', 'calendar_read'])",
				},
			},
			"required": []string{"title", "type", "schedule", "prompt_condition"},
		},
	}
}

func (t *CreateTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	title, _ := input["title"].(string)
	taskTypeStr, _ := input["type"].(string)
	schedule, _ := input["schedule"].(string)
	promptCondition, _ := input["prompt_condition"].(string)
	tz, _ := input["timezone"].(string)
	targetConnector, _ := input["target_connector"].(string)
	targetChannelID, _ := input["target_channel_id"].(string)

	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(schedule) == "" {
		return nil, fmt.Errorf("schedule is required")
	}
	if strings.TrimSpace(promptCondition) == "" {
		return nil, fmt.Errorf("prompt_condition is required")
	}

	taskType := domain.TaskType(strings.ToLower(strings.TrimSpace(taskTypeStr)))
	if taskType != domain.TaskTypeCron && taskType != domain.TaskTypeWatch {
		return nil, fmt.Errorf("type must be 'cron' or 'watch'")
	}

	cfgTz := ""
	if t.cfg != nil {
		cfgTz = t.cfg.App.Timezone
	}
	loc := scheduler.ResolveTimezone(tz, cfgTz)

	now := time.Now()
	var nextRun time.Time

	if taskType == domain.TaskTypeWatch {
		mins, err := strconv.Atoi(strings.TrimSpace(schedule))
		if err != nil {
			return nil, fmt.Errorf("schedule for watch must be an integer minute string, got: %s", schedule)
		}
		if mins < 5 {
			return nil, fmt.Errorf("watch interval cannot be less than 5 minutes (rate limit safeguard)")
		}
		nextRun = now.Add(time.Duration(mins) * time.Minute)
	} else {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		sched, err := parser.Parse(schedule)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression %q: %w. Expected 5 fields: 'minute hour day-of-month month day-of-week'", schedule, err)
		}
		nowInLoc := now.In(loc)
		nextRun = sched.Next(nowInLoc).UTC()
	}

	var targetTools []string
	if rawTools, ok := input["target_tools"].([]interface{}); ok {
		for _, item := range rawTools {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				targetTools = append(targetTools, strings.TrimSpace(s))
			}
		}
	}

	connectorType, _ := input["connector_type"].(string)
	if connectorType == "" {
		connectorType = "web"
	}
	channelID, _ := input["channel_id"].(string)
	if channelID == "" {
		channelID = "web"
	}
	sessionID, _ := input["session_id"].(string)
	if sessionID == "" {
		sessionID = "default"
	}

	if targetConnector == "" {
		targetConnector = connectorType
	}
	if targetChannelID == "" {
		targetChannelID = channelID
	}

	task := &domain.ProactiveTask{
		ID:              uuid.New().String(),
		SessionID:       sessionID,
		ConnectorType:   connectorType,
		ChannelID:       channelID,
		TargetConnector: targetConnector,
		TargetChannelID: targetChannelID,
		Title:           title,
		TaskType:        taskType,
		ScheduleExpr:    schedule,
		Timezone:        loc.String(),
		PromptCondition: promptCondition,
		TargetTools:     targetTools,
		IsActive:        true,
		NextRunAt:       nextRun,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := t.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("save proactive task: %w", err)
	}

	return fmt.Sprintf("✅ Proactive task '%s' created successfully!\n- Type: %s\n- Schedule: %s (%s)\n- Target: %s (%s)\n- Next execution: %s",
		task.Title, task.TaskType, task.ScheduleExpr, task.Timezone, task.TargetConnector, task.TargetChannelID, task.NextRunAt.In(loc).Format("2006-01-02 15:04:05 MST")), nil
}
