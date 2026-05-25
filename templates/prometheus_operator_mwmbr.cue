package templates

import (
	"strings"
)

#Alert: {
	long:     string
	short:    string
	burn:     float
	severity: "page" | "message" | "ticket"
}

#Input: {
	slo_target:           float
	error_budget_ratio:   float
	total_window_seconds: float
	windows: [...#Alert]
	config?: {[string]: _}
}

// Data input
data: #Input

// Helper to extract config with defaults
_config: {
	metric_name: *data.config.metric_name | "slo_errors"
	namespace:   *data.config.namespace | "monitoring"
	runbook_url: *data.config.runbook_url | ""
	rule_labels: *data.config.rule_labels | {}
	meta_labels: *data.config.meta_labels | {"role": "alert-rules"}
}

// Helper to filter and format alerts
#FormatAlerts: {
	sev: string
	_match: [for a in data.windows if a.severity == sev {a}]
	_list: [for a in _match {
		"(job:\(_config.metric_name):rate\(a.long) > (\(a.burn) * \(data.error_budget_ratio)) and job:\(_config.metric_name):rate\(a.short) > (\(a.burn) * \(data.error_budget_ratio)))"
	}]
	exists: len(_list) > 0
	out:    strings.Join(_list, "\n or \n")
	
	// Annotations building
	summary: "High burn rate for \(sev) alerts"
	description: "The \(sev) error budget is burning too fast for metric \(_config.metric_name). Current burn rate exceeds SRE thresholds."
}

_pages:    #FormatAlerts & {sev: "page"}
_messages: #FormatAlerts & {sev: "message"}
_tickets:  #FormatAlerts & {sev: "ticket"}

// The final PrometheusRule object
apiVersion: "monitoring.coreos.com/v1"
kind:       "PrometheusRule"
metadata: {
	name:      [if _config.meta_labels.name != _|_ { _config.meta_labels.name }, "slo-alerts"][0]
	namespace: _config.namespace
	labels:    _config.meta_labels
}
spec: {
	groups: [
		{
			name: "slo-alerts"
			rules: [
				if _pages.exists {
					alert: "HighBurnRatePage"
					labels: _config.rule_labels & {severity: "page"}
					expr: _pages.out
					annotations: {
						summary:     _pages.summary
						description: _pages.description
						if _config.runbook_url != "" {
							runbook_url: _config.runbook_url
						}
					}
				},
				if _messages.exists {
					alert: "HighBurnRateMessage"
					labels: _config.rule_labels & {severity: "message"}
					expr: _messages.out
					annotations: {
						summary:     _messages.summary
						description: _messages.description
						if _config.runbook_url != "" {
							runbook_url: _config.runbook_url
						}
					}
				},
				if _tickets.exists {
					alert: "HighBurnRateTicket"
					labels: _config.rule_labels & {severity: "ticket"}
					expr: _tickets.out
					annotations: {
						summary:     _tickets.summary
						description: _tickets.description
						if _config.runbook_url != "" {
							runbook_url: _config.runbook_url
						}
					}
				},
			]
		},
	]
}
