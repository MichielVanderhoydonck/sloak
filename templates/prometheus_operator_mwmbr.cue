package templates

// Data input
data: #Input

_meta: {
	namespace:   *data.config.namespace | "monitoring"
	meta_labels: *data.config.meta_labels | {"role": "alert-rules"}
}

_ruleGroups: #MWMBRRuleGroups & {"data": data}

// The final PrometheusRule object
apiVersion: "monitoring.coreos.com/v1"
kind:       "PrometheusRule"
metadata: {
	name:      *data.config.meta_labels.name | "slo-alerts"
	namespace: _meta.namespace
	labels:    _meta.meta_labels
}
spec: {
	groups: _ruleGroups.groups
}
