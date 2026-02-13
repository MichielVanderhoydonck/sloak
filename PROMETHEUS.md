# 🧪 End-to-End Prometheus Verification

To validate that sloak generates valid configuration that correctly triggers alerts, you can run a local test lab using Minikube and the Prometheus Operator.
## 1. Start the Lab Environment

Spin up a local cluster and install the standard kube-prometheus stack.
```Bash

minikube start
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace
```
## 2. Inject Mock Data (The Fixture)

Instead of generating fake traffic, we inject Recording Rules that force specific metric values. This deterministically simulates an outage (e.g., a 20% error rate).

Save this as fixtures.yaml:
```YAML

apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: slo-fixtures
  namespace: monitoring
  labels:
    release: prometheus # Required for Operator to pick it up
spec:
  groups:
    - name: mock-data-generator
      rules:
        # Simulate a 20% error rate (0.2) across all windows
        # This > 14.4x burn rate, triggering Page, Message, and Ticket alerts.
        - record: job:api_errors:rate5m
          expr: vector(0.2)
        - record: job:api_errors:rate30m
          expr: vector(0.2)
        - record: job:api_errors:rate1h
          expr: vector(0.2)
        - record: job:api_errors:rate6h
          expr: vector(0.2)
        - record: job:api_errors:rate3d
          expr: vector(0.2)
```
Apply it: `kubectl apply -f fixtures.yaml`
## 3. Generate & Apply Alerts

Run sloak to generate the alerting rules.
Important: We add the release=prometheus label so the Operator loads the file, and set the window to 0s to skip the "Pending" wait time for instant verification.
Bash

### Generate the YAML
```
./sloak generate prometheus \
  --slo 99.9 \
  --metric-name api_errors \
  --rule-labels "service=api,env=prod" \
  --meta-labels "release=prometheus" \
  --namespace monitoring \
  --window 30d > slo-alerts.yaml
```
### (Optional) Edit yaml to set 'for: 0s' for instant firing
`sed -i 's/for: 5m/for: 0s/g' slo-alerts.yaml`

### Apply to Cluster
`kubectl apply -f slo-alerts.yaml`

## 4. Verify in UI

Port-forward the Prometheus UI:
```Bash

kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring
```
Visit http://localhost:9090/alerts. You should see HighBurnRatePage, HighBurnRateMessage, and HighBurnRateTicket in the Firing (Red) state.
