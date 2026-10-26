package templates

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
