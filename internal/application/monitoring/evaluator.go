package monitoring

import "github.com/domainry/domainry-monitoring-sdk/contract"

func EvaluateHealth(input contract.HealthRequest) map[string]any {
	status := "ok"
	checks := map[string]string{"storage": "ok", "migration": "ok", "scheduler": "ok", "lifecycle": "ok"}
	warnings := map[string]any{}
	if input.Storage.Error != "" {
		status, checks["storage"] = "degraded", "error"
	}
	if input.Migration.Error != "" {
		status, checks["migration"] = "degraded", "error"
	} else if !input.Migration.Current {
		status, checks["migration"] = "degraded", "outdated"
	}
	if input.Scheduler.Error != "" {
		status, checks["scheduler"] = "degraded", "error"
	} else if value := schedulerWarnings(input.Scheduler.Payload); len(value) > 0 {
		status, checks["scheduler"], warnings["scheduler"] = "degraded", "warning", value
	}
	if input.Lifecycle.Error != "" {
		status, checks["lifecycle"] = "degraded", "error"
	} else if warning, _ := input.Lifecycle.Payload["warning"].(bool); warning {
		status, checks["lifecycle"], warnings["lifecycle"] = "degraded", "warning", input.Lifecycle.Payload
	}
	payload := map[string]any{
		"status": status, "runtime_id": input.RuntimeID,
		"template_id": input.Identity.TemplateID, "template_version": input.Identity.TemplateVersion,
		"checks": checks, "storage": input.Storage.Payload, "migration": input.Migration.Payload,
		"scheduler": input.Scheduler.Payload, "lifecycle": input.Lifecycle.Payload,
	}
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	return payload
}

func EvaluateMetrics(input contract.MetricsRequest) map[string]any {
	result := map[string]any{"runtime_id": input.RuntimeID, "template_id": input.Identity.TemplateID, "template_version": input.Identity.TemplateVersion}
	for key, value := range input.Sections {
		result[key] = value
	}
	if len(input.Errors) > 0 {
		result["errors"] = input.Errors
	}
	return result
}

func schedulerWarnings(status map[string]any) map[string]any {
	warnings := map[string]any{}
	available, _ := status["runtime_available"].(bool)
	if !available {
		return warnings
	}
	if value := integer(status["unresolved_dead_letters"]); value > 0 {
		warnings["unresolved_dead_letters"] = value
	}
	if value := integer(status["lease_expirations"]); value > 0 {
		warnings["stale_leased_runs"] = value
	}
	return warnings
}

func integer(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
