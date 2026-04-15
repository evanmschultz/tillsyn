package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/config"
)

// embeddingsStatusCommandOptions stores embeddings status flag values.
type embeddingsStatusCommandOptions struct {
	projectID       string
	crossProject    bool
	includeArchived bool
	statuses        []string
	limit           int
}

// embeddingsReindexCommandOptions stores embeddings reindex flag values.
type embeddingsReindexCommandOptions struct {
	projectID        string
	crossProject     bool
	includeArchived  bool
	force            bool
	wait             bool
	waitTimeout      time.Duration
	waitPollInterval time.Duration
}

// buildEmbeddingRuntimeConfig converts config-file settings into app runtime settings.
func buildEmbeddingRuntimeConfig(cfg config.Config, appName, command string) (app.EmbeddingRuntimeConfig, error) {
	pollInterval, err := parseEmbeddingDuration("embeddings.worker_poll_interval", cfg.Embeddings.WorkerPollInterval)
	if err != nil {
		return app.EmbeddingRuntimeConfig{}, err
	}
	claimTTL, err := parseEmbeddingDuration("embeddings.claim_ttl", cfg.Embeddings.ClaimTTL)
	if err != nil {
		return app.EmbeddingRuntimeConfig{}, err
	}
	initialRetryBackoff, err := parseEmbeddingDuration("embeddings.initial_retry_backoff", cfg.Embeddings.InitialRetryBackoff)
	if err != nil {
		return app.EmbeddingRuntimeConfig{}, err
	}
	maxRetryBackoff, err := parseEmbeddingDuration("embeddings.max_retry_backoff", cfg.Embeddings.MaxRetryBackoff)
	if err != nil {
		return app.EmbeddingRuntimeConfig{}, err
	}

	command = firstNonEmpty(command, "tui")
	return app.EmbeddingRuntimeConfig{
		Enabled:             cfg.Embeddings.Enabled,
		Provider:            cfg.Embeddings.Provider,
		Model:               cfg.Embeddings.Model,
		BaseURL:             cfg.Embeddings.BaseURL,
		Dimensions:          cfg.Embeddings.Dimensions,
		ModelSignature:      app.BuildEmbeddingModelSignature(cfg.Embeddings.Provider, cfg.Embeddings.Model, cfg.Embeddings.BaseURL, cfg.Embeddings.Dimensions),
		MaxAttempts:         cfg.Embeddings.MaxAttempts,
		PollInterval:        pollInterval,
		ClaimTTL:            claimTTL,
		InitialRetryBackoff: initialRetryBackoff,
		MaxRetryBackoff:     maxRetryBackoff,
		WorkerID:            buildEmbeddingWorkerID(appName, command),
	}.Normalize(), nil
}

// runEmbeddingsStatus renders the operator-facing embeddings lifecycle inventory.
func runEmbeddingsStatus(ctx context.Context, svc *app.Service, opts embeddingsStatusCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	projectIDs, err := resolveEmbeddingProjectScope(ctx, svc, opts.projectID, opts.crossProject, opts.includeArchived, "embeddings status")
	if err != nil {
		return err
	}
	statuses, err := parseEmbeddingStatuses(opts.statuses)
	if err != nil {
		return err
	}

	rows, err := svc.ListEmbeddingStates(ctx, app.EmbeddingListFilter{
		ProjectIDs: projectIDs,
		Statuses:   statuses,
		Limit:      opts.limit,
	})
	if err != nil {
		return fmt.Errorf("list embedding states: %w", err)
	}
	summary, err := svc.SummarizeEmbeddingStates(ctx, app.EmbeddingListFilter{
		ProjectIDs: projectIDs,
	})
	if err != nil {
		return fmt.Errorf("summarize embedding states: %w", err)
	}
	return writeEmbeddingStatus(stdout, svc.EmbeddingsOperational(), projectIDs, summary, rows)
}

// runEmbeddingsReindex triggers one explicit reindex/backfill request and reports progress.
func runEmbeddingsReindex(ctx context.Context, svc *app.Service, cfg config.Config, opts embeddingsReindexCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if !cfg.Embeddings.Enabled {
		return fmt.Errorf("embeddings are disabled; enable embeddings.enabled before running reindex")
	}
	if !opts.crossProject && strings.TrimSpace(opts.projectID) == "" {
		return fmt.Errorf("--project-id is required for embeddings reindex unless --cross-project is set")
	}

	ctx = cliMutationContext(ctx, cfg)
	result, err := svc.ReindexEmbeddings(ctx, app.ReindexEmbeddingsInput{
		ProjectID:        strings.TrimSpace(opts.projectID),
		CrossProject:     opts.crossProject,
		IncludeArchived:  opts.includeArchived,
		Force:            opts.force,
		Wait:             opts.wait,
		WaitTimeout:      opts.waitTimeout,
		WaitPollInterval: opts.waitPollInterval,
	})
	if err != nil {
		return fmt.Errorf("reindex embeddings: %w", err)
	}
	return writeEmbeddingReindexResult(stdout, result)
}

// resolveEmbeddingProjectScope expands one CLI scope selection into concrete project ids.
func resolveEmbeddingProjectScope(ctx context.Context, svc *app.Service, projectID string, crossProject, includeArchived bool, command string) ([]string, error) {
	if crossProject {
		projects, err := svc.ListProjects(ctx, includeArchived)
		if err != nil {
			return nil, fmt.Errorf("%s: list projects: %w", command, err)
		}
		out := make([]string, 0, len(projects))
		for _, project := range projects {
			out = append(out, project.ID)
		}
		return out, nil
	}

	if err := requireProjectID(command, projectID); err != nil {
		return nil, err
	}
	project, err := locateProjectForCLI(ctx, svc, projectID, includeArchived, command)
	if err != nil {
		return nil, err
	}
	return []string{project.ID}, nil
}

// parseEmbeddingStatuses validates one CLI lifecycle status filter list.
func parseEmbeddingStatuses(values []string) ([]app.EmbeddingLifecycleStatus, error) {
	out := make([]app.EmbeddingLifecycleStatus, 0, len(values))
	for _, value := range values {
		switch strings.TrimSpace(strings.ToLower(value)) {
		case "":
		case string(app.EmbeddingLifecyclePending):
			out = append(out, app.EmbeddingLifecyclePending)
		case string(app.EmbeddingLifecycleRunning):
			out = append(out, app.EmbeddingLifecycleRunning)
		case string(app.EmbeddingLifecycleReady):
			out = append(out, app.EmbeddingLifecycleReady)
		case string(app.EmbeddingLifecycleFailed):
			out = append(out, app.EmbeddingLifecycleFailed)
		case string(app.EmbeddingLifecycleStale):
			out = append(out, app.EmbeddingLifecycleStale)
		default:
			return nil, fmt.Errorf("unsupported embeddings status %q; allowed values: pending, running, ready, failed, stale", value)
		}
	}
	return out, nil
}

// writeEmbeddingStatus renders one summary-first inventory for operators.
func writeEmbeddingStatus(stdout io.Writer, runtimeOperational bool, projectIDs []string, summary app.EmbeddingSummary, rows []app.EmbeddingRecord) error {
	runtimeLabel := "unavailable"
	if runtimeOperational {
		runtimeLabel = "operational"
	}
	if err := writeCLIKV(stdout, "Embeddings Status", [][2]string{
		{"runtime", runtimeLabel},
		{"projects", embeddingScopeLabel(projectIDs)},
		{"pending", fmt.Sprintf("%d", summary.PendingCount)},
		{"running", fmt.Sprintf("%d", summary.RunningCount)},
		{"ready", fmt.Sprintf("%d", summary.ReadyCount)},
		{"failed", fmt.Sprintf("%d", summary.FailedCount)},
		{"stale", fmt.Sprintf("%d", summary.StaleCount)},
	}); err != nil {
		return err
	}
	if len(rows) == 0 {
		return writeCLIPanel(
			stdout,
			"Embeddings Inventory",
			"No matching embedding lifecycle rows.",
			"Next step: till embeddings reindex --project-id PROJECT_ID --wait",
		)
	}

	renderRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		renderRows = append(renderRows, []string{
			row.ProjectID,
			string(row.SubjectType),
			row.SubjectID,
			string(row.Status),
			formatEmbeddingTime(row.UpdatedAt),
			formatEmbeddingTimePtr(row.NextAttemptAt),
			formatEmbeddingTimePtr(row.LastSucceededAt),
			embeddingDetail(row),
		})
	}
	if err := writeCLITable(
		stdout,
		"Embeddings Inventory",
		[]string{"PROJECT", "TYPE", "SUBJECT", "STATUS", "UPDATED", "NEXT ATTEMPT", "LAST SUCCESS", "DETAIL"},
		renderRows,
		"No matching embedding lifecycle rows.",
	); err != nil {
		return err
	}
	return writeCLIPanel(
		stdout,
		"Health Guidance",
		"Healthy means ready > 0 with pending/running/failed/stale all at 0 for the chosen scope.",
		"",
	)
}

// writeEmbeddingReindexResult renders one explicit reindex outcome.
func writeEmbeddingReindexResult(stdout io.Writer, result app.ReindexEmbeddingsResult) error {
	status := "queued"
	if result.TimedOut {
		status = "timed_out"
	} else if result.FailedCount > 0 && result.PendingCount == 0 && result.RunningCount == 0 && result.StaleCount == 0 {
		status = "failed"
	} else if result.Completed {
		status = "completed"
	}
	if err := writeCLIKV(stdout, "Embeddings Reindex", [][2]string{
		{"status", status},
		{"target projects", embeddingScopeLabel(result.TargetProjects)},
		{"scanned", fmt.Sprintf("%d", result.ScannedCount)},
		{"queued", fmt.Sprintf("%d", result.QueuedCount)},
		{"ready", fmt.Sprintf("%d", result.ReadyCount)},
		{"running", fmt.Sprintf("%d", result.RunningCount)},
		{"failed", fmt.Sprintf("%d", result.FailedCount)},
		{"stale", fmt.Sprintf("%d", result.StaleCount)},
		{"pending", fmt.Sprintf("%d", result.PendingCount)},
	}); err != nil {
		return fmt.Errorf("write embeddings reindex output: %w", err)
	}

	body := "Embedding lifecycle is steady for the requested scope."
	footer := ""
	switch {
	case result.TimedOut:
		body = "Next step: rerun till embeddings status to inspect pending, failed, or stale rows."
	case result.FailedCount > 0 && result.PendingCount == 0 && result.RunningCount == 0 && result.StaleCount == 0:
		body = "Embedding lifecycle reached a terminal failed state for part of the requested scope."
		footer = "Next step: till embeddings status --status failed"
	case !result.Completed:
		body = "Next step: use till embeddings status to watch the background worker settle."
	}
	return writeCLIPanel(stdout, "Embeddings Reindex Guidance", body, footer)
}

// embeddingScopeLabel converts a project id list into one readable scope label.
func embeddingScopeLabel(projectIDs []string) string {
	if len(projectIDs) == 0 {
		return "(none)"
	}
	return strings.Join(projectIDs, ", ")
}

// embeddingDetail condenses one lifecycle row into the operator-facing detail column.
func embeddingDetail(row app.EmbeddingRecord) string {
	switch {
	case strings.TrimSpace(row.StaleReason) != "":
		return row.StaleReason
	case strings.TrimSpace(row.LastErrorSummary) != "":
		return row.LastErrorSummary
	case strings.TrimSpace(row.ModelSignature) != "":
		return row.ModelSignature
	default:
		return "-"
	}
}

// formatEmbeddingTime renders one required timestamp consistently for CLI status output.
func formatEmbeddingTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Local().Format(time.RFC3339)
}

// formatEmbeddingTimePtr renders one optional timestamp consistently for CLI status output.
func formatEmbeddingTimePtr(ts *time.Time) string {
	if ts == nil {
		return "-"
	}
	return formatEmbeddingTime(*ts)
}

// parseEmbeddingDuration parses one embeddings runtime duration setting.
func parseEmbeddingDuration(label, raw string) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("%s is required", label)
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", label, raw, err)
	}
	return duration, nil
}

// buildEmbeddingWorkerID creates one process-scoped worker identifier that is stable for the life of the process.
func buildEmbeddingWorkerID(appName, command string) string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn("resolve hostname for embeddings worker failed", "err", err)
		hostname = "unknown-host"
	}
	return strings.Join([]string{
		sanitizeWorkerSegment(firstNonEmpty(appName, "tillsyn")),
		sanitizeWorkerSegment(firstNonEmpty(command, "tui")),
		sanitizeWorkerSegment(hostname),
		fmt.Sprintf("pid-%d", os.Getpid()),
	}, ":")
}

// sanitizeWorkerSegment normalizes one worker-id segment into a log-safe token.
func sanitizeWorkerSegment(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	var out strings.Builder
	out.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '.', r == '-', r == '_':
			out.WriteRune(r)
		default:
			out.WriteByte('-')
		}
	}
	return strings.Trim(out.String(), "-")
}
