// Package handlers provides HTTP handler functions for the Bruce API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bruce/internal/repository"
)

// logsResponse is the JSON body returned by the /api/v1/logs endpoint.
type logsResponse struct {
	Logs []logEntryResponse `json:"logs"`
}

type logEntryResponse struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Logger    string                 `json:"logger"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
}

// LogsHandler godoc
// @Summary      Get application logs
// @Description  Retrieve application logs with optional filtering by level, logger, time range, and limit
// @Tags         monitoring
// @Produce      json
// @Param        level       query   string  false  "Log level (e.g., INFO, DEBUG, ERROR)"
// @Param        logger      query   string  false  "Logger name to filter by"
// @Param        start_time  query   string  false  "Start time in ISO8601 format (e.g., 2024-01-01T00:00:00Z)"
// @Param        end_time    query   string  false  "End time in ISO8601 format (e.g., 2024-01-01T23:59:59Z)"
// @Param        limit       query   int     false  "Maximum number of logs to return (default: no limit)"
// @Success      200         {object}  logsResponse
// @Failure      400         {object}  map[string]string  "Invalid query parameters"
// @Failure      500         {object}  map[string]string  "Internal server error"
// @Router       /api/v1/logs [get]
func LogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		monitoringRepo, ok := r.Context().Value("monitoringRepo").(*repository.MonitoringRepository)
		if !ok {
			http.Error(w, "monitoring repository not available", http.StatusInternalServerError)
			return
		}

		// Parse query parameters
		levelStr := r.URL.Query().Get("level")
		loggerStr := r.URL.Query().Get("logger")
		startTimeStr := r.URL.Query().Get("start_time")
		endTimeStr := r.URL.Query().Get("end_time")
		limitStr := r.URL.Query().Get("limit")

		opts := repository.LogQueryOptions{}

		if levelStr != "" {
			opts.Level = &levelStr
		}
		if loggerStr != "" {
			opts.Logger = &loggerStr
		}

		if startTimeStr != "" {
			t, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid start_time format: %v", err), http.StatusBadRequest)
				return
			}
			opts.StartTime = &t
		}

		if endTimeStr != "" {
			t, err := time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid end_time format: %v", err), http.StatusBadRequest)
				return
			}
			opts.EndTime = &t
		}

		if limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "invalid limit: must be an integer", http.StatusBadRequest)
				return
			}
			opts.Limit = limit
		}

		logs, err := monitoringRepo.GetLogs(ctx, opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to retrieve logs: %v", err), http.StatusInternalServerError)
			return
		}

		// Convert to response format
		logResponses := make([]logEntryResponse, len(logs))
		for i, log := range logs {
			logResponses[i] = logEntryResponse{
				Timestamp: log.Timestamp.Format(time.RFC3339),
				Level:     string(log.Level),
				Logger:    log.Logger,
				Message:   log.Message,
				Context:   log.Context,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(logsResponse{Logs: logResponses})
	}
}

// metricsResponse is the JSON body returned by the /api/v1/metrics endpoint.
type metricsResponse struct {
	Metrics []metricGroupResponse `json:"metrics"`
}

type metricGroupResponse struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Datapoints []metricDatapointResponse `json:"datapoints"`
}

type metricDatapointResponse struct {
	Timestamp string            `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// MetricsHandler godoc
// @Summary      Get application metrics
// @Description  Retrieve application metrics with optional filtering by metric name, time range, and aggregation window
// @Tags         monitoring
// @Produce      json
// @Param        metric_name          query   string  false  "Metric name to filter by (e.g., http_requests_total)"
// @Param        start_time           query   string  false  "Start time in ISO8601 format (e.g., 2024-01-01T00:00:00Z)"
// @Param        end_time             query   string  false  "End time in ISO8601 format (e.g., 2024-01-01T23:59:59Z)"
// @Param        aggregation_window   query   int     false  "Aggregation window in seconds for metric data"
// @Success      200                  {object}  metricsResponse
// @Failure      400                  {object}  map[string]string  "Invalid query parameters"
// @Failure      500                  {object}  map[string]string  "Internal server error"
// @Router       /api/v1/metrics [get]
func MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		monitoringRepo, ok := r.Context().Value("monitoringRepo").(*repository.MonitoringRepository)
		if !ok {
			http.Error(w, "monitoring repository not available", http.StatusInternalServerError)
			return
		}

		// Parse query parameters
		metricNameStr := r.URL.Query().Get("metric_name")
		startTimeStr := r.URL.Query().Get("start_time")
		endTimeStr := r.URL.Query().Get("end_time")
		aggregationStr := r.URL.Query().Get("aggregation_window")

		opts := repository.MetricQueryOptions{}

		if metricNameStr != "" {
			opts.MetricName = &metricNameStr
		}

		if startTimeStr != "" {
			t, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid start_time format: %v", err), http.StatusBadRequest)
				return
			}
			opts.StartTime = &t
		}

		if endTimeStr != "" {
			t, err := time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid end_time format: %v", err), http.StatusBadRequest)
				return
			}
			opts.EndTime = &t
		}

		if aggregationStr != "" {
			agg, err := strconv.Atoi(aggregationStr)
			if err != nil {
				http.Error(w, "invalid aggregation_window: must be an integer", http.StatusBadRequest)
				return
			}
			opts.AggregationWindow = agg
		}

		metrics, err := monitoringRepo.GetMetrics(ctx, opts)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to retrieve metrics: %v", err), http.StatusInternalServerError)
			return
		}

		// Group metrics by name and type
		metricGroups := make(map[string]metricGroupResponse)
		for _, metric := range metrics {
			key := metric.MetricName
			group, exists := metricGroups[key]
			if !exists {
				group = metricGroupResponse{
					Name:       metric.MetricName,
					Type:       string(metric.MetricType),
					Datapoints: make([]metricDatapointResponse, 0),
				}
			}
			group.Datapoints = append(group.Datapoints, metricDatapointResponse{
				Timestamp: metric.Timestamp.Format(time.RFC3339),
				Value:     metric.Value,
				Labels:    metric.Labels,
			})
			metricGroups[key] = group
		}

		// Convert to response format
		metricResponses := make([]metricGroupResponse, 0, len(metricGroups))
		for _, group := range metricGroups {
			metricResponses = append(metricResponses, group)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(metricsResponse{Metrics: metricResponses})
	}
}

// monitoringConfigResponse represents the current monitoring configuration.
type monitoringConfigResponse struct {
	LogEnabled     bool   `json:"log_enabled"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	RetentionDays  int    `json:"retention_days"`
}

// MonitoringConfigGetHandler godoc
// @Summary      Get monitoring configuration
// @Description  Retrieve the current monitoring configuration settings
// @Tags         monitoring
// @Produce      json
// @Success      200  {object}  monitoringConfigResponse
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /api/v1/monitoring/config [get]
func MonitoringConfigGetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		monitoringRepo, ok := r.Context().Value("monitoringRepo").(*repository.MonitoringRepository)
		if !ok {
			http.Error(w, "monitoring repository not available", http.StatusInternalServerError)
			return
		}

		config, err := monitoringRepo.GetAllConfig(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to retrieve config: %v", err), http.StatusInternalServerError)
			return
		}

		// Parse config values with defaults
		logEnabled := true
		metricsEnabled := true
		retentionDays := 30

		if v, exists := config["log_enabled"]; exists && v == "false" {
			logEnabled = false
		}
		if v, exists := config["metrics_enabled"]; exists && v == "false" {
			metricsEnabled = false
		}
		if v, exists := config["retention_days"]; exists {
			if days, err := strconv.Atoi(v); err == nil {
				retentionDays = days
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(monitoringConfigResponse{
			LogEnabled:     logEnabled,
			MetricsEnabled: metricsEnabled,
			RetentionDays:  retentionDays,
		})
	}
}

// MonitoringConfigPatchHandler godoc
// @Summary      Update monitoring configuration
// @Description  Update monitoring configuration settings (admin-only, should be protected by middleware)
// @Tags         monitoring
// @Produce      json
// @Accept       json
// @Param        body  body      monitoringConfigResponse  true  "Monitoring configuration to update"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string  "Invalid request body"
// @Failure      500   {object}  map[string]string  "Internal server error"
// @Router       /api/v1/monitoring/config [patch]
func MonitoringConfigPatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		monitoringRepo, ok := r.Context().Value("monitoringRepo").(*repository.MonitoringRepository)
		if !ok {
			http.Error(w, "monitoring repository not available", http.StatusInternalServerError)
			return
		}

		var req monitoringConfigResponse
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Update config values
		if err := monitoringRepo.SetConfig(ctx, "log_enabled", fmt.Sprintf("%v", req.LogEnabled)); err != nil {
			http.Error(w, fmt.Sprintf("failed to update log_enabled: %v", err), http.StatusInternalServerError)
			return
		}

		if err := monitoringRepo.SetConfig(ctx, "metrics_enabled", fmt.Sprintf("%v", req.MetricsEnabled)); err != nil {
			http.Error(w, fmt.Sprintf("failed to update metrics_enabled: %v", err), http.StatusInternalServerError)
			return
		}

		if err := monitoringRepo.SetConfig(ctx, "retention_days", fmt.Sprintf("%d", req.RetentionDays)); err != nil {
			http.Error(w, fmt.Sprintf("failed to update retention_days: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}
