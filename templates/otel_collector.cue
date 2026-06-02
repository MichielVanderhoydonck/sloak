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
	total_window_seconds: float
	windows: [...#Alert]
	config?: {[string]: _}
}

// Data input
data: #Input

// Configuration helpers
_config: {
	service_name: *data.config.service_name | "my-service"
	receiver:     *data.config.receiver | "otlp"
	exporter:     *data.config.exporter | "prometheus"
}

receivers: {
	"\(_config.receiver)": {
		protocols: {
			grpc: {}
			http: {}
		}
	}
}

connectors: {
	spanmetrics: {
		histogram: {
			explicit: {
				buckets: ["10ms", "50ms", "100ms", "250ms", "500ms", "1s", "2s", "5s", "10s"]
			}
		}
		dimensions: [
			{name: "http.method"},
			{name: "http.status_code"},
		]
	}
}

exporters: {
	"\(_config.exporter)": {
		if _config.exporter == "prometheus" {
			endpoint: "0.0.0.0:8889"
		}
	}
}

processors: {
	batch: {}
}

service: {
	pipelines: {
		traces: {
			receivers: [_config.receiver]
			processors: ["batch"]
			exporters: ["spanmetrics"]
		}
		metrics: {
			receivers: ["spanmetrics"]
			processors: ["batch"]
			exporters: [_config.exporter]
		}
	}
}

