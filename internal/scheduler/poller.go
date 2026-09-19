// Package scheduler provides background polling and execution scheduling for proactive tasks.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"

	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/worker"
)

// Poller periodically evaluates due proactive tasks from SQLite and enqueues Asynq jobs.
type Poller struct {
	repo        repository.ProactiveTaskRepository
	asynqClient *asynq.Client
	cfg         *config.Config
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
	cronParser  cron.Parser
}

// NewPoller returns a new Poller configured with a 60-second evaluation interval.
func NewPoller(
	repo repository.ProactiveTaskRepository,
	asynqClient *asynq.Client,
	cfg *config.Config,
) *Poller {
	return &Poller{
		repo:        repo,
		asynqClient: asynqClient,
		cfg:         cfg,
		interval:    60 * time.Second,
		stopCh:      make(chan struct{}),
		cronParser:  cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

// SetInterval overrides the ticker interval (primarily for testing).
func (p *Poller) SetInterval(d time.Duration) {
	p.interval = d
}

// Start launches the background polling loop.
func (p *Poller) Start(ctx context.Context) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		log.Printf("INFO: scheduler poller started (interval: %v)", p.interval)

		p.EvaluateDueTasks(ctx)

		for {
			select {
			case <-ticker.C:
				p.EvaluateDueTasks(ctx)
			case <-p.stopCh:
				log.Println("INFO: scheduler poller stopped")
				return
			case <-ctx.Done():
				log.Println("INFO: scheduler poller context canceled")
				return
			}
		}
	}()
}

// Stop gracefully shuts down the poller.
func (p *Poller) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

// ResolveTimezone resolves the IANA location for a task based on:
// 1. Task-specified timezone
// 2. Host machine local timezone (if valid and not UTC/Local)
// 3. Config App.Timezone
// 4. Fallback to "America/Sao_Paulo"
func ResolveTimezone(taskTimezone string, cfgTimezone string) *time.Location {
	// 1. Task explicit timezone
	if strings.TrimSpace(taskTimezone) != "" {
		if loc, err := time.LoadLocation(taskTimezone); err == nil {
			return loc
		}
	}

	// 2. Host local timezone if valid and distinct
	if loc := time.Local; loc != nil {
		name := loc.String()
		if name != "" && name != "UTC" && name != "Local" {
			return loc
		}
	}

	// 3. Config timezone
	if strings.TrimSpace(cfgTimezone) != "" {
		if loc, err := time.LoadLocation(cfgTimezone); err == nil {
			return loc
		}
	}

	// 4. Default fallback to São Paulo
	if loc, err := time.LoadLocation("America/Sao_Paulo"); err == nil {
		return loc
	}

	return time.UTC
}

// CalculateNextRun computes the next execution time for a proactive task.
func (p *Poller) CalculateNextRun(task *domain.ProactiveTask, now time.Time) (time.Time, error) {
	cfgTz := ""
	if p.cfg != nil {
		cfgTz = p.cfg.App.Timezone
	}
	loc := ResolveTimezone(task.Timezone, cfgTz)

	switch task.TaskType {
	case domain.TaskTypeWatch:
		mins, err := strconv.Atoi(strings.TrimSpace(task.ScheduleExpr))
		if err != nil || mins < 5 {
			mins = 5 // enforce 5 minute minimum rate limit
		}
		return now.Add(time.Duration(mins) * time.Minute), nil

	case domain.TaskTypeCron:
		sched, err := p.cronParser.Parse(task.ScheduleExpr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", task.ScheduleExpr, err)
		}
		nowInLoc := now.In(loc)
		nextInLoc := sched.Next(nowInLoc)
		return nextInLoc.UTC(), nil

	default:
		return now.Add(15 * time.Minute), nil
	}
}

// EvaluateDueTasks queries SQLite for due tasks and dispatches them to Asynq.
func (p *Poller) EvaluateDueTasks(ctx context.Context) {
	now := time.Now()
	tasks, err := p.repo.GetDueTasks(ctx, now)
	if err != nil {
		log.Printf("ERROR: scheduler poller get due tasks: %v", err)
		return
	}

	for _, task := range tasks {
		nextRun, err := p.CalculateNextRun(&task, now)
		if err != nil {
			log.Printf("ERROR: calculate next run for task %s (%s): %v", task.ID, task.Title, err)
			continue
		}

		// Update database first to prevent double-queueing
		if err := p.repo.UpdateNextRun(ctx, task.ID, now, nextRun); err != nil {
			log.Printf("ERROR: update next run for task %s (%s): %v", task.ID, task.Title, err)
			continue
		}

		if p.asynqClient == nil {
			continue
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
				LastResultHash:  task.LastResultHash,
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
		}

		if err != nil {
			log.Printf("ERROR: create asynq task for %s (%s): %v", task.ID, task.Title, err)
			continue
		}

		if _, err := p.asynqClient.Enqueue(asynqTask); err != nil {
			log.Printf("ERROR: enqueue proactive task %s (%s) to redis: %v", task.ID, task.Title, err)
		} else {
			log.Printf("INFO: enqueued proactive task %s (%s, type=%s, next_run=%v)", task.ID, task.Title, task.TaskType, nextRun)
		}
	}
}
