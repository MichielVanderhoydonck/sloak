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

#Config: {
	metric_name: string
	rule_labels: [string]: string
	meta_labels: [string]: string
	namespace:   string
	runbook_url: string
}

#Input: {
	budget: float
	alerts: [...#Alert]
	config: #Config
}

// Data input
data: #Input

// Helper to filter and format alerts
#FormatAlerts: {
	sev: string
	_match: [for a in data.alerts if a.severity == sev {a}]
	_list: [for a in _match {
		"(job:\(data.config.metric_name):rate\(a.long) > (\(a.burn) * \(data.budget)) and job:\(data.config.metric_name):rate\(a.short) > (\(a.burn) * \(data.budget)))"
	}]
	exists: len(_list) > 0
	out:    strings.Join(_list, "\n or \n")
	
	// Annotations building
	summary: "High burn rate for \(sev) alerts"
	description: "The \(sev) error budget is burning too fast for metric \(data.config.metric_name). Current burn rate exceeds SRE thresholds."
}

_pages:    #FormatAlerts & {sev: "page"}
_messages: #FormatAlerts & {sev: "message"}
_tickets:  #FormatAlerts & {sev: "ticket"}

// The final PrometheusRule object
apiVersion: "monitoring.coreos.com/v1"
kind:       "PrometheusRule"
metadata: {
	name:      [if data.config.meta_labels.name != _|_ { data.config.meta_labels.name }, "slo-alerts"][0]
	namespace: data.config.namespace
	labels:    data.config.meta_labels
}
spec: {
	groups: [
		{
			name: "slo-alerts"
			rules: [
				if _pages.exists {
					alert: "HighBurnRatePage"
					labels: data.config.rule_labels & {severity: "page"}
					expr: _pages.out
					annotations: {
						summary:     _pages.summary
						description: _pages.description
						if data.config.runbook_url != "" {
							runbook_url: data.config.runbook_url
						}
					}
				},
				if _messages.exists {
					alert: "HighBurnRateMessage"
					labels: data.config.rule_labels & {severity: "message"}
					expr: _messages.out
					annotations: {
						summary:     _messages.summary
						description: _messages.description
						if data.config.runbook_url != "" {
							runbook_url: data.config.runbook_url
						}
					}
				},
				if _tickets.exists {
					alert: "HighBurnRateTicket"
					labels: data.config.rule_labels & {severity: "ticket"}
					expr: _tickets.out
					annotations: {
						summary:     _tickets.summary
						description: _tickets.description
						if data.config.runbook_url != "" {
							runbook_url: data.config.runbook_url
						}
					}
				},
			]
		},
	]
}
