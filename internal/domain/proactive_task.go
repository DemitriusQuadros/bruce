package domain

import (
	"encoding/json"
	"time"
)

// TaskType distinguishes between interval condition watches and scheduled cron reports.
type TaskType string

const (
	TaskTypeWatch TaskType = "watch"
	TaskTypeCron  TaskType = "cron"
)

// ProactiveTask represents an ongoing background monitoring watch or scheduled report.
type ProactiveTask struct {
	ID              string     `json:"id"`
	SessionID       string     `json:"session_id"`
	ConnectorType   string     `json:"connector_type"`
	ChannelID       string     `json:"channel_id"`
	TargetConnector string     `json:"target_connector"`
	TargetChannelID string     `json:"target_channel_id"`
	Title           string     `json:"title"`
	TaskType        TaskType   `json:"task_type"`
	ScheduleExpr    string     `json:"schedule_expr"`
	Timezone        string     `json:"timezone"`
	PromptCondition string     `json:"prompt_condition"`
	TargetTools     []string   `json:"target_tools"`
	IsActive        bool       `json:"is_active"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastResultHash  string     `json:"last_result_hash"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TargetToolsJSON returns TargetTools serialized as a JSON string for SQLite storage.
func (t *ProactiveTask) TargetToolsJSON() string {
	if len(t.TargetTools) == 0 {
		return "[]"
	}
	b, err := json.Marshal(t.TargetTools)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// SetTargetToolsFromJSON parses a JSON string from SQLite into TargetTools.
func (t *ProactiveTask) SetTargetToolsFromJSON(data string) {
	if data == "" || data == "[]" {
		t.TargetTools = []string{}
		return
	}
	var tools []string
	if err := json.Unmarshal([]byte(data), &tools); err != nil {
		t.TargetTools = []string{}
		return
	}
	t.TargetTools = tools
}
