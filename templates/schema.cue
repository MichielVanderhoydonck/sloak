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
