package templates

import (
	"strings"
)

// Data input
data: #Input

// Helper to extract config with defaults
_config: {
	metric_name:  *data.config.metric_name | "slo_errors"
	namespace:    *data.config.namespace | "monitoring"
	runbook_url:  *data.config.runbook_url | ""
	rule_labels:  *data.config.rule_labels | {}
	meta_labels:  *data.config.meta_labels | {"role": "alert-rules"}
	errors_query: *data.config.errors_query | ""
	total_query:  *data.config.total_query | ""
	sli_query:    *data.config.sli_query | ""

	rule_annotations: *data.config.rule_annotations | {}

	severity_labels: {
		page:    *(data.config.severity_labels.page) | {}
		message: *(data.config.severity_labels.message) | {}
		ticket:  *(data.config.severity_labels.ticket) | {}
	}

	severity_annotations: {
		page:    *(data.config.severity_annotations.page) | {}
		message: *(data.config.severity_annotations.message) | {}
		ticket:  *(data.config.severity_annotations.ticket) | {}
	}
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
	
	labels: _config.rule_labels & _config.severity_labels[sev] & {severity: sev}

	annotations: _config.rule_annotations & _config.severity_annotations[sev] & {
		summary:     "High burn rate for \(sev) alerts"
		description: "The \(sev) error budget is burning too fast for metric \(_config.metric_name). Current burn rate exceeds SRE thresholds."
		if _config.runbook_url != "" {
			runbook_url: _config.runbook_url
		}
	}
}

_pages:    #FormatAlerts & {sev: "page"}
_messages: #FormatAlerts & {sev: "message"}
_tickets:  #FormatAlerts & {sev: "ticket"}

#FormatQueryPart: {
	q: string
	w: string
	out: string
	if strings.Contains(q, "$window") {
		out: strings.Replace(q, "$window", w, -1)
	}
	if strings.Contains(q, "{{window}}") {
		out: strings.Replace(q, "{{window}}", w, -1)
	}
	if strings.Contains(q, "[window]") {
		out: strings.Replace(q, "[window]", "[\(w)]", -1)
	}
	if !strings.Contains(q, "$window") && !strings.Contains(q, "{{window}}") && !strings.Contains(q, "[window]") {
		if strings.HasPrefix(q, "vector(") {
			out: q
		}
		if !strings.HasPrefix(q, "vector(") {
			if strings.Contains(q, "[") {
				out: q
			}
			if !strings.Contains(q, "[") {
				out: "sum(rate(\(q)[\(w)]))"
			}
		}
	}
}

#FormatRecordingExpr: {
	w: string
	out: string
	if _config.sli_query != "" {
		_sli: #FormatQueryPart & {q: _config.sli_query, "w": w}
		out: _sli.out
	}
	if _config.sli_query == "" && _config.errors_query != "" {
		_err: #FormatQueryPart & {q: _config.errors_query, "w": w}
		if _config.total_query != "" {
			_tot: #FormatQueryPart & {q: _config.total_query, "w": w}
			out: "\(_err.out) / \(_tot.out)"
		}
		if _config.total_query == "" {
			out: _err.out
		}
	}
	if _config.sli_query == "" && _config.errors_query == "" {
		out: ""
	}
}

_has_recordings: (_config.errors_query != "" || _config.sli_query != "") && (data.recording_windows != _|_) && len(data.recording_windows) > 0

// The final PrometheusRule object
apiVersion: "monitoring.coreos.com/v1"
kind:       "PrometheusRule"
metadata: {
	name:      *data.config.meta_labels.name | "slo-alerts"
	namespace: _config.namespace
	labels:    _config.meta_labels
}
spec: {
	groups: [
		if _has_recordings {
			name: "slo-recording-rules"
			rules: [
				for w in data.recording_windows {
					record: "job:\(_config.metric_name):rate\(w)"
					expr:   (#FormatRecordingExpr & {"w": w}).out
					if len(_config.rule_labels) > 0 {
						labels: _config.rule_labels
					}
				},
			]
		},
		{
			name: "slo-alerts"
			rules: [
				if _pages.exists {
					alert:       "HighBurnRatePage"
					labels:      _pages.labels
					expr:        _pages.out
					annotations: _pages.annotations
				},
				if _messages.exists {
					alert:       "HighBurnRateMessage"
					labels:      _messages.labels
					expr:        _messages.out
					annotations: _messages.annotations
				},
				if _tickets.exists {
					alert:       "HighBurnRateTicket"
					labels:      _tickets.labels
					expr:        _tickets.out
					annotations: _tickets.annotations
				},
			]
		},
	]
}
