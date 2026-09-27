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
	repo        repository.ProactiveTaskRepository
	sessionRepo repository.SessionRepository
	cfg         *config.Config
}

// NewCreateTool returns a new CreateTool.
func NewCreateTool(repo repository.ProactiveTaskRepository, sessionRepo repository.SessionRepository, cfg *config.Config) *CreateTool {
	return &CreateTool{repo: repo, sessionRepo: sessionRepo, cfg: cfg}
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
					"description": "For cron with mode 'message': the exact text of the message/reminder to send directly. For cron with mode 'agent': instructions on what report to compile. For watch: natural language trigger condition",
				},
				"execution_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"message", "agent"},
					"description": "Optional: 'message' to deliver the text directly as a notification/reminder without LLM reprocessing (best for simple reminders like 'take medicine', 'hello world'); 'agent' (default) to run the prompt through the AI agent to search/use tools and compile a dynamic report",
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
	execMode, _ := input["execution_mode"].(string)
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

	execMode = strings.ToLower(strings.TrimSpace(execMode))
	if execMode != "message" && execMode != "agent" {
		lowerPrompt := strings.ToLower(strings.TrimSpace(promptCondition))
		if strings.HasPrefix(lowerPrompt, "send the message:") ||
			strings.HasPrefix(lowerPrompt, "send message:") ||
			strings.HasPrefix(lowerPrompt, "enviar mensagem:") ||
			strings.HasPrefix(lowerPrompt, "enviar a mensagem:") ||
			strings.HasPrefix(lowerPrompt, "lembrete:") ||
			strings.HasPrefix(lowerPrompt, "reminder:") ||
			strings.Contains(strings.ToLower(title), "hello world") {
			execMode = "message"
		} else {
			execMode = "agent"
		}
	}
	if execMode == "message" {
		promptCondition = cleanDirectMessageText(promptCondition)
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
		nowInLoc := now.In(loc)
		parsedExpr, parsedNext, err := parseCronSchedule(schedule, nowInLoc, parser)
		if err != nil {
			return nil, err
		}
		schedule = parsedExpr
		nextRun = parsedNext
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
	channelID, _ := input["channel_id"].(string)
	sessionID, _ := input["session_id"].(string)

	// Auto-detect connector and channel from context if not explicitly provided
	if ctxConn, ctxChan, ok := ai.ConnectorFromContext(ctx); ok {
		if connectorType == "" {
			connectorType = ctxConn
		}
		if channelID == "" {
			channelID = ctxChan
		}
	}
	if connectorType == "" {
		connectorType = "web"
	}
	if channelID == "" {
		channelID = "web"
	}

	// Auto-detect session ID from context if not provided
	if sessionID == "" {
		if ctxSess, ok := ai.SessionIDFromContext(ctx); ok && ctxSess != "" {
			sessionID = ctxSess
		}
	}

	// Ensure session exists in DB to satisfy foreign key constraints
	if t.sessionRepo != nil {
		if sessionID != "" && sessionID != "default" {
			existing, err := t.sessionRepo.GetByID(sessionID)
			if err != nil || existing == nil {
				sess, err := t.sessionRepo.FindOrCreate(connectorType, channelID)
				if err == nil && sess != nil {
					sessionID = sess.ID
				}
			}
		} else {
			sess, err := t.sessionRepo.FindOrCreate(connectorType, channelID)
			if err == nil && sess != nil {
				sessionID = sess.ID
			}
		}
	}
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
		ExecutionMode:   execMode,
		IsActive:        true,
		NextRunAt:       nextRun,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := t.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("save proactive task: %w", err)
	}

	return fmt.Sprintf("✅ Proactive task '%s' created successfully!\n- Type: %s (mode: %s)\n- Schedule: %s (%s)\n- Target: %s (%s)\n- Next execution: %s",
		task.Title, task.TaskType, task.ExecutionMode, task.ScheduleExpr, task.Timezone, task.TargetConnector, task.TargetChannelID, task.NextRunAt.In(loc).Format("2006-01-02 15:04:05 MST")), nil
}

// parseCronSchedule parses either standard 5-token cron syntax or relative offsets (e.g. "+2m", "2m", "in 2 minutes", "daqui a 2 minutos").
func parseCronSchedule(schedule string, nowInLoc time.Time, parser cron.Parser) (string, time.Time, error) {
	trimmed := strings.ToLower(strings.TrimSpace(schedule))

	// Check if this is a relative offset
	cleaned := strings.TrimPrefix(trimmed, "+")
	cleaned = strings.TrimPrefix(cleaned, "in ")
	cleaned = strings.TrimPrefix(cleaned, "daqui a ")
	cleaned = strings.TrimSuffix(cleaned, " minutes")
	cleaned = strings.TrimSuffix(cleaned, " minutos")
	cleaned = strings.TrimSuffix(cleaned, " mins")
	cleaned = strings.TrimSuffix(cleaned, " min")
	cleaned = strings.TrimSuffix(cleaned, " m")
	cleaned = strings.TrimSpace(cleaned)

	var duration time.Duration
	var isRelative bool

	if d, err := time.ParseDuration(cleaned); err == nil && d > 0 {
		duration = d
		isRelative = true
	} else if d, err := time.ParseDuration(cleaned + "m"); err == nil && d > 0 {
		duration = d
		isRelative = true
	} else if mins, err := strconv.Atoi(cleaned); err == nil && mins > 0 {
		duration = time.Duration(mins) * time.Minute
		isRelative = true
	}

	if isRelative {
		target := nowInLoc.Add(duration)
		cronExpr := fmt.Sprintf("%d %d %d %d *", target.Minute(), target.Hour(), target.Day(), int(target.Month()))
		return cronExpr, target.UTC(), nil
	}

	sched, err := parser.Parse(schedule)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid cron expression %q: %w. Expected 5 fields ('minute hour day-of-month month day-of-week') or relative offset (e.g. '+2m', '10m')", schedule, err)
	}
	return schedule, sched.Next(nowInLoc).UTC(), nil
}

// cleanDirectMessageText removes common command prefixes and enclosing quotes from direct messages.
func cleanDirectMessageText(text string) string {
	trimmed := strings.TrimSpace(text)
	prefixes := []string{
		"send the message:",
		"send message:",
		"send:",
		"enviar a mensagem:",
		"enviar mensagem:",
		"enviar:",
		"lembrete:",
		"reminder:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			trimmed = strings.TrimSpace(trimmed[len(prefix):])
			break
		}
	}

	if (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2) ||
		(strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") && len(trimmed) >= 2) {
		trimmed = trimmed[1 : len(trimmed)-1]
	}

	return strings.TrimSpace(trimmed)
}
