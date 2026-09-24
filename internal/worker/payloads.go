// Package worker defines Asynq task payload types, task constructors, and task name constants.
package worker

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// TaskProcessIncomingMessage is the Asynq task type for incoming connector messages.
const TaskProcessIncomingMessage = "message:process"

// ProcessIncomingMessagePayload is the JSON payload for a message:process task.
type ProcessIncomingMessagePayload struct {
	ConnectorType string `json:"connector_type"` // "whatsapp" | "discord"
	ChannelID     string `json:"channel_id"`     // phone number or Discord channel ID
	Content       string `json:"content"`        // raw text from the user
}

// NewProcessIncomingMessageTask creates an Asynq task for processing an incoming message.
// MaxRetry is 3; Timeout is 90s (must exceed Claude's 60s HTTP timeout).
func NewProcessIncomingMessageTask(p ProcessIncomingMessagePayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskProcessIncomingMessage, payload,
		asynq.MaxRetry(3),
		asynq.Timeout(90*time.Second),
	), nil
}

// TaskEvaluateWatch is the Asynq task type for evaluating condition-based ambient watches.
const TaskEvaluateWatch = "proactive:evaluate_watch"

// TaskExecuteScheduledReport is the Asynq task type for generating scheduled time-of-day reports.
const TaskExecuteScheduledReport = "proactive:execute_report"

// EvaluateWatchPayload is the JSON payload for a proactive:evaluate_watch task.
type EvaluateWatchPayload struct {
	TaskID          string   `json:"task_id"`
	SessionID       string   `json:"session_id"`
	TargetConnector string   `json:"target_connector"`
	TargetChannelID string   `json:"target_channel_id"`
	Title           string   `json:"title"`
	Condition       string   `json:"condition"`
	TargetTools     []string `json:"target_tools"`
	LastResultHash  string   `json:"last_result_hash"`
}

// NewEvaluateWatchTask creates an Asynq task for evaluating a proactive watch.
func NewEvaluateWatchTask(p EvaluateWatchPayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskEvaluateWatch, payload,
		asynq.MaxRetry(2),
		asynq.Timeout(45*time.Second),
	), nil
}

// ExecuteScheduledReportPayload is the JSON payload for a proactive:execute_report task.
type ExecuteScheduledReportPayload struct {
	TaskID          string `json:"task_id"`
	SessionID       string `json:"session_id"`
	TargetConnector string `json:"target_connector"`
	TargetChannelID string `json:"target_channel_id"`
	Title           string `json:"title"`
	Prompt          string `json:"prompt"`
}

// NewExecuteScheduledReportTask creates an Asynq task for executing a scheduled report.
func NewExecuteScheduledReportTask(p ExecuteScheduledReportPayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExecuteScheduledReport, payload,
		asynq.MaxRetry(2),
		asynq.Timeout(120*time.Second),
	), nil
}

// TaskSummarizeSession is the Asynq task type for background conversation summarization.
const TaskSummarizeSession = "session:summarize"

// SummarizeSessionPayload is the JSON payload for a session:summarize task.
type SummarizeSessionPayload struct {
	SessionID string `json:"session_id"`
}

// NewSummarizeSessionTask creates an Asynq task for summarizing older messages in a session.
func NewSummarizeSessionTask(p SummarizeSessionPayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskSummarizeSession, payload,
		asynq.MaxRetry(2),
		asynq.Timeout(60*time.Second),
	), nil
}
