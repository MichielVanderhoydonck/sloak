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
	total_window_seconds: int | float
	windows: [...#Alert]
	recording_windows?: [...string]
	config?: {[string]: _}
}

#FormatQueryPart: {
	q:  string
	w:  string
	by: string
	_hasPlaceholder: strings.Contains(q, "$window") || strings.Contains(q, "{{window}}") || strings.Contains(q, "[window]")
	_isVector:       strings.HasPrefix(q, "vector(")

	_sub1: strings.Replace(q, "$window", w, -1)
	_sub2: strings.Replace(_sub1, "{{window}}", w, -1)
	_sub3: strings.Replace(_sub2, "[window]", "[\(w)]", -1)

	_sumPrefix: [
		if by != "" && by != "none" { "sum by (\(by)) " },
		"sum",
	][0]

	out: [
		if _hasPlaceholder { _sub3 },
		if _isVector { q },
		"\(_sumPrefix)(rate(\(q)[\(w)]))",
	][0]
}

#FormatRatioQueryPart: {
	q:  string
	w:  string
	by: string
	_hasPlaceholder: strings.Contains(q, "$window") || strings.Contains(q, "{{window}}") || strings.Contains(q, "[window]")
	_isVector:       strings.HasPrefix(q, "vector(")

	_sub1: strings.Replace(q, "$window", w, -1)
	_sub2: strings.Replace(_sub1, "{{window}}", w, -1)
	_sub3: strings.Replace(_sub2, "[window]", "[\(w)]", -1)

	out: [
		if _hasPlaceholder { _sub3 },
		if _isVector { q },
		if by != "" && by != "none" { "avg by (\(by)) (avg_over_time(\(q)[\(w)]))" },
		"avg_over_time(\(q)[\(w)])",
	][0]
}

#FormatRecordingExpr: {
	w:                  string
	error_ratio_query:  string
	sli_query:          string
	errors_query:       string
	total_query:        string
	aggregation_labels: string

	_ratio_q: [
		if error_ratio_query != "" { error_ratio_query },
		sli_query,
	][0]

	_by: aggregation_labels

	out: string
	if _ratio_q != "" {
		_ratio: #FormatRatioQueryPart & {q: _ratio_q, "w": w, by: _by}
		out: _ratio.out
	}
	if _ratio_q == "" && errors_query != "" {
		_err: #FormatQueryPart & {q: errors_query, "w": w, by: _by}
		if total_query != "" {
			_tot: #FormatQueryPart & {q: total_query, "w": w, by: _by}
			out: "\(_err.out) / \(_tot.out)"
		}
		if total_query == "" {
			out: _err.out
		}
	}
	if _ratio_q == "" && errors_query == "" {
		out: ""
	}
}

#FormatAlerts: {
	sev:                  string
	windows:              [...#Alert]
	budget_ratio:         float
	metric_name:          string
	runbook_url:          string
	rule_labels:          {[string]: _}
	rule_annotations:     {[string]: _}
	severity_labels:      {[string]: _}
	severity_annotations: {[string]: _}

	_match: [for a in windows if a.severity == sev {a}]
	_list: [for a in _match {
		"(job:\(metric_name):rate\(a.long) > (\(a.burn) * \(budget_ratio)) and job:\(metric_name):rate\(a.short) > (\(a.burn) * \(budget_ratio)))"
	}]
	exists: len(_list) > 0
	out:    strings.Join(_list, "\n or \n")

	labels: rule_labels & severity_labels & {severity: sev}

	let _rb = runbook_url
	annotations: rule_annotations & severity_annotations & {
		summary:     "High burn rate for \(sev) alerts"
		description: "The \(sev) error budget is burning too fast for metric \(metric_name). Current burn rate exceeds SRE thresholds."
		if _rb != "" {
			runbook_url: _rb
		}
	}
}

#MWMBRConfig: {
	data: #Input

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

#MWMBRRuleGroups: {
	data: #Input
	cfg:  #MWMBRConfig & {"data": data}

	_alerts: {
		sev: string
		#FormatAlerts & {
			"sev":                sev
			windows:              data.windows
			budget_ratio:         data.error_budget_ratio
			metric_name:          cfg.metric_name
			runbook_url:          cfg.runbook_url
			rule_labels:          cfg.rule_labels
			rule_annotations:     cfg.rule_annotations
			severity_labels:      cfg.severity_labels[sev]
			severity_annotations: cfg.severity_annotations[sev]
		}
	}

	_pages:    _alerts & {sev: "page"}
	_messages: _alerts & {sev: "message"}
	_tickets:  _alerts & {sev: "ticket"}

	_has_recordings: (cfg.errors_query != "" || cfg.error_ratio_query != "" || cfg.sli_query != "") && (data.recording_windows != _|_) && len(data.recording_windows) > 0

	groups: [
		if _has_recordings {
			name: "slo-recording-rules"
			rules: [
				for w in data.recording_windows {
					record: "job:\(cfg.metric_name):rate\(w)"
					expr:   (#FormatRecordingExpr & {
						"w":                w
						error_ratio_query:  cfg.error_ratio_query
						sli_query:          cfg.sli_query
						errors_query:       cfg.errors_query
						total_query:        cfg.total_query
						aggregation_labels: cfg.aggregation_labels
					}).out
					if len(cfg.rule_labels) > 0 {
						labels: cfg.rule_labels
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
