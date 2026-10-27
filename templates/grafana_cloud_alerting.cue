package templates

// Data input
data: #Input

// Helper to extract config with defaults
_config: {
	metric_name:        *data.config.metric_name | "slo_errors"
	runbook_url:        *data.config.runbook_url | ""
	rule_labels:        *data.config.rule_labels | {}
	errors_query:       *data.config.errors_query | ""
	total_query:        *data.config.total_query | ""
	_raw_ratio: [
		if data.config.error_ratio_query != _|_ { data.config.error_ratio_query },
		if data.config.sli_query != _|_ { data.config.sli_query },
		"",
	][0]
	error_ratio_query:  _raw_ratio
	sli_query:          _raw_ratio
	aggregation_labels: *data.config.aggregation_labels | "job"

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

// Alert rule formatters using shared helper
_alerts: {
	sev: string
	#FormatAlerts & {
		"sev":                 sev
		windows:              data.windows
		budget_ratio:         data.error_budget_ratio
		metric_name:          _config.metric_name
		runbook_url:          _config.runbook_url
		rule_labels:          _config.rule_labels
		rule_annotations:     _config.rule_annotations
		severity_labels:      _config.severity_labels[sev]
		severity_annotations: _config.severity_annotations[sev]
	}
}

_pages:    _alerts & {sev: "page"}
_messages: _alerts & {sev: "message"}
_tickets:  _alerts & {sev: "ticket"}

_has_recordings: (_config.errors_query != "" || _config.error_ratio_query != "" || _config.sli_query != "") && (data.recording_windows != _|_) && len(data.recording_windows) > 0

// Standard Mimir/Loki ruler format
groups: [
	if _has_recordings {
		name: "slo-recording-rules"
		rules: [
			for w in data.recording_windows {
				record: "job:\(_config.metric_name):rate\(w)"
				expr:   (#FormatRecordingExpr & {
					"w":                 w
					error_ratio_query:  _config.error_ratio_query
					sli_query:          _config.sli_query
					errors_query:       _config.errors_query
					total_query:        _config.total_query
					aggregation_labels: _config.aggregation_labels
				}).out
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
