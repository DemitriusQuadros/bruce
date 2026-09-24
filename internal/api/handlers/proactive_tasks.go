package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"

	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/scheduler"
	"bruce/internal/worker"
)

// ProactiveTasksHandler handles CRUD operations and manual triggers for proactive tasks.
func ProactiveTasksHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := r.Context().Value("proactiveTaskRepo").(repository.ProactiveTaskRepository)
		if !ok || repo == nil {
			writeError(w, http.StatusInternalServerError, "proactive task repository not initialized")
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		switch r.Method {
		case http.MethodGet:
			if id == "" {
				handleListProactiveTasks(w, r, repo)
			} else {
				handleGetProactiveTask(w, r, repo, id)
			}
		case http.MethodPost:
			if id == "" {
				handleCreateProactiveTask(w, r, repo, cfg)
			} else {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		case http.MethodPatch:
			handlePatchProactiveTask(w, r, repo, id, cfg)
		case http.MethodDelete:
			handleDeleteProactiveTask(w, r, repo, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// ProactiveTaskRunHandler immediately enqueues a proactive task execution for manual testing.
func ProactiveTaskRunHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := r.Context().Value("proactiveTaskRepo").(repository.ProactiveTaskRepository)
		if !ok || repo == nil {
			writeError(w, http.StatusInternalServerError, "proactive task repository not initialized")
			return
		}

		asynqClient, ok := r.Context().Value("asynqClient").(*asynq.Client)
		if !ok || asynqClient == nil {
			writeError(w, http.StatusInternalServerError, "asynq client not initialized")
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		task, err := repo.GetByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query task: "+err.Error())
			return
		}
		if task == nil {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		var asynqTask *asynq.Task
		switch task.TaskType {
		case domain.TaskTypeWatch:
			asynqTask, err = worker.NewEvaluateWatchTask(worker.EvaluateWatchPayload{
				TaskID:          task.ID,
				SessionID:       task.SessionID,
				TargetConnector: task.TargetConnector,
				TargetChannelID: task.TargetChannelID,
				Title:           task.Title,
				Condition:       task.PromptCondition,
				TargetTools:     task.TargetTools,
				LastResultHash:  "", // bypass deduplication for manual run
			})
		case domain.TaskTypeCron:
			asynqTask, err = worker.NewExecuteScheduledReportTask(worker.ExecuteScheduledReportPayload{
				TaskID:          task.ID,
				SessionID:       task.SessionID,
				TargetConnector: task.TargetConnector,
				TargetChannelID: task.TargetChannelID,
				Title:           task.Title,
				Prompt:          task.PromptCondition,
			})
		default:
			writeError(w, http.StatusBadRequest, "unknown task type")
			return
		}

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create task payload: "+err.Error())
			return
		}

		if _, err := asynqClient.Enqueue(asynqTask); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to enqueue task: "+err.Error())
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"message": "execution enqueued successfully",
			"task_id": task.ID,
		})
	}
}

func handleListProactiveTasks(w http.ResponseWriter, r *http.Request, repo repository.ProactiveTaskRepository) {
	sessionID := r.URL.Query().Get("session_id")
	var tasks []domain.ProactiveTask
	var err error

	if strings.TrimSpace(sessionID) != "" {
		tasks, err = repo.ListBySession(r.Context(), sessionID)
	} else {
		tasks, err = repo.ListAll(r.Context())
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list proactive tasks: "+err.Error())
		return
	}

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		if expectedActive, parseErr := strconv.ParseBool(isActiveStr); parseErr == nil {
			var filtered []domain.ProactiveTask
			for _, t := range tasks {
				if t.IsActive == expectedActive {
					filtered = append(filtered, t)
				}
			}
			tasks = filtered
		}
	}

	if tasks == nil {
		tasks = []domain.ProactiveTask{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func handleGetProactiveTask(w http.ResponseWriter, r *http.Request, repo repository.ProactiveTaskRepository, id string) {
	task, err := repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get proactive task: "+err.Error())
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type createProactiveTaskRequest struct {
	SessionID       string   `json:"session_id"`
	ConnectorType   string   `json:"connector_type"`
	ChannelID       string   `json:"channel_id"`
	TargetConnector string   `json:"target_connector"`
	TargetChannelID string   `json:"target_channel_id"`
	Title           string   `json:"title"`
	TaskType        string   `json:"task_type"`
	ScheduleExpr    string   `json:"schedule_expr"`
	Timezone        string   `json:"timezone"`
	PromptCondition string   `json:"prompt_condition"`
	TargetTools     []string `json:"target_tools"`
}

func handleCreateProactiveTask(w http.ResponseWriter, r *http.Request, repo repository.ProactiveTaskRepository, cfg *config.Config) {
	var req createProactiveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if strings.TrimSpace(req.ScheduleExpr) == "" {
		writeError(w, http.StatusBadRequest, "schedule_expr is required")
		return
	}
	if strings.TrimSpace(req.PromptCondition) == "" {
		writeError(w, http.StatusBadRequest, "prompt_condition is required")
		return
	}

	taskType := domain.TaskType(strings.ToLower(strings.TrimSpace(req.TaskType)))
	if taskType != domain.TaskTypeCron && taskType != domain.TaskTypeWatch {
		writeError(w, http.StatusBadRequest, "task_type must be 'cron' or 'watch'")
		return
	}

	cfgTz := ""
	if cfg != nil {
		cfgTz = cfg.App.Timezone
	}
	loc := scheduler.ResolveTimezone(req.Timezone, cfgTz)

	now := time.Now()
	var nextRun time.Time

	if taskType == domain.TaskTypeWatch {
		mins, err := strconv.Atoi(strings.TrimSpace(req.ScheduleExpr))
		if err != nil {
			writeError(w, http.StatusBadRequest, "schedule_expr for watch must be an integer minute string")
			return
		}
		if mins < 5 {
			writeError(w, http.StatusBadRequest, "watch interval cannot be less than 5 minutes")
			return
		}
		nextRun = now.Add(time.Duration(mins) * time.Minute)
	} else {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		sched, err := parser.Parse(req.ScheduleExpr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cron expression: "+err.Error())
			return
		}
		nowInLoc := now.In(loc)
		nextRun = sched.Next(nowInLoc).UTC()
	}

	connType := req.ConnectorType
	if connType == "" {
		connType = "web"
	}
	chanID := req.ChannelID
	if chanID == "" {
		chanID = "web"
	}

	sessionID := req.SessionID
	sessionRepo, ok := r.Context().Value("sessionRepo").(repository.SessionRepository)
	if ok && sessionRepo != nil {
		if sessionID != "" && sessionID != "default" {
			existing, err := sessionRepo.GetByID(sessionID)
			if err == nil && existing != nil {
				sessionID = existing.ID
			} else {
				sess, err := sessionRepo.FindOrCreate(connType, chanID)
				if err == nil && sess != nil {
					sessionID = sess.ID
				}
			}
		} else {
			sess, err := sessionRepo.FindOrCreate(connType, chanID)
			if err == nil && sess != nil {
				sessionID = sess.ID
			}
		}
	}
	if sessionID == "" {
		sessionID = "default"
	}

	targetConn := req.TargetConnector
	if targetConn == "" {
		targetConn = connType
	}
	targetChan := req.TargetChannelID
	if targetChan == "" {
		targetChan = chanID
	}

	task := &domain.ProactiveTask{
		ID:              uuid.New().String(),
		SessionID:       sessionID,
		ConnectorType:   connType,
		ChannelID:       chanID,
		TargetConnector: targetConn,
		TargetChannelID: targetChan,
		Title:           req.Title,
		TaskType:        taskType,
		ScheduleExpr:    req.ScheduleExpr,
		Timezone:        loc.String(),
		PromptCondition: req.PromptCondition,
		TargetTools:     req.TargetTools,
		IsActive:        true,
		NextRunAt:       nextRun,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := repo.Create(r.Context(), task); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

type patchProactiveTaskRequest struct {
	IsActive        *bool     `json:"is_active"`
	Title           *string   `json:"title"`
	ScheduleExpr    *string   `json:"schedule_expr"`
	PromptCondition *string   `json:"prompt_condition"`
	Timezone        *string   `json:"timezone"`
	TargetConnector *string   `json:"target_connector"`
	TargetChannelID *string   `json:"target_channel_id"`
	TargetTools     *[]string `json:"target_tools"`
}

func handlePatchProactiveTask(w http.ResponseWriter, r *http.Request, repo repository.ProactiveTaskRepository, id string, cfg *config.Config) {
	task, err := repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query task: "+err.Error())
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	var req patchProactiveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	scheduleChanged := false
	if req.IsActive != nil {
		task.IsActive = *req.IsActive
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		task.Title = strings.TrimSpace(*req.Title)
	}
	if req.PromptCondition != nil && strings.TrimSpace(*req.PromptCondition) != "" {
		task.PromptCondition = strings.TrimSpace(*req.PromptCondition)
	}
	if req.Timezone != nil && strings.TrimSpace(*req.Timezone) != "" {
		task.Timezone = strings.TrimSpace(*req.Timezone)
		scheduleChanged = true
	}
	if req.ScheduleExpr != nil && strings.TrimSpace(*req.ScheduleExpr) != "" {
		task.ScheduleExpr = strings.TrimSpace(*req.ScheduleExpr)
		scheduleChanged = true
	}
	if req.TargetConnector != nil && strings.TrimSpace(*req.TargetConnector) != "" {
		targetConn := strings.ToLower(strings.TrimSpace(*req.TargetConnector))
		if targetConn != "whatsapp" && targetConn != "discord" && targetConn != "telegram" && targetConn != "web" {
			writeError(w, http.StatusBadRequest, "target_connector must be 'whatsapp', 'discord', 'telegram', or 'web'")
			return
		}
		task.TargetConnector = targetConn
	}
	if req.TargetChannelID != nil {
		task.TargetChannelID = strings.TrimSpace(*req.TargetChannelID)
	}
	if req.TargetTools != nil {
		task.TargetTools = *req.TargetTools
	}

	if scheduleChanged {
		cfgTz := ""
		if cfg != nil {
			cfgTz = cfg.App.Timezone
		}
		loc := scheduler.ResolveTimezone(task.Timezone, cfgTz)
		task.Timezone = loc.String()
		now := time.Now()

		if task.TaskType == domain.TaskTypeWatch {
			mins, err := strconv.Atoi(task.ScheduleExpr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "schedule_expr for watch must be an integer minute string")
				return
			}
			if mins < 5 {
				writeError(w, http.StatusBadRequest, "watch interval cannot be less than 5 minutes")
				return
			}
			task.NextRunAt = now.Add(time.Duration(mins) * time.Minute)
		} else {
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
			sched, err := parser.Parse(task.ScheduleExpr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid cron expression: "+err.Error())
				return
			}
			task.NextRunAt = sched.Next(now.In(loc)).UTC()
		}
	}

	task.UpdatedAt = time.Now()
	if err := repo.Update(r.Context(), task); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func handleDeleteProactiveTask(w http.ResponseWriter, r *http.Request, repo repository.ProactiveTaskRepository, id string) {
	if err := repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "task deleted"})
}
