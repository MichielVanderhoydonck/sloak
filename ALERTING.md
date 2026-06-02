# 🧪 Prometheus Operator Verification

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

### Render the YAML
```
./sloak render \
  --template prometheus-operator \
  --slo 99.9 \
  --set metric_name=api_errors \
  --set "rule_labels=service=api,env=prod" \
  --set "meta_labels=release=prometheus" \
  --set namespace=monitoring \
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

---

# 📊 Grafana Cloud Alerting Verification

Grafana Cloud Alerting uses standard Mimir/Loki ruler format rules. Grafana Cloud offers a generous **Free Tier** to host rules.

## 1. Syntax Validation
You can validate the generated alerting rules syntax locally using the official `mimirtool` CLI utility without creating an account:

```bash
# Generate the rules file
./sloak render --template grafana-cloud --slo 99.9 --window 30d --values templates/values.yaml > mimir-alerts.yaml

# Validate syntax (exit code 0 indicates success)
mimirtool rules check mimir-alerts.yaml

# Lint/format rules (sorts keys alphabetically and formats expressions to a single line)
mimirtool rules lint mimir-alerts.yaml

# Analyze used metrics (generates metrics-in-ruler.json listing all metric names referenced in the rules)
mimirtool analyze rule-file mimir-alerts.yaml
```

## 2. Local Live Testing via Docker
You can run a lightweight Grafana Mimir instance locally using Docker to test real rule synchronization:

1. Spin up a local Mimir instance with filesystem ruler storage enabled:
```bash
docker run -d --name sloak-mimir -p 9009:8080 grafana/mimir:latest -ruler-storage.backend=filesystem -ruler-storage.local.directory=/tmp/mimir-ruler
```

2. Sync the generated rules to the local Mimir ruler (we use `anonymous` tenant ID):
```bash
mimirtool rules sync --address=http://localhost:9009 --id=anonymous mimir-alerts.yaml
```

3. Verify that the rules are loaded into the Mimir ruler:
```bash
mimirtool rules print --address=http://localhost:9009 --id=anonymous
```

4. Tear down the local Mimir instance:
```bash
docker rm -f sloak-mimir
```

---

# 🚀 OpenTelemetry Collector Verification

Verify that the generated configuration starts up and parses correctly.

## 1. Syntax & Startup Validation (Free & Local)
Run the OpenTelemetry Collector container locally using Docker to test if the generated configuration is valid:

```bash
# Generate the OTel configuration file
./sloak render --template otel-collector --slo 99.9 --window 30d --values templates/values.yaml > otel-collector.yaml

# Validation via Docker using 'validate' subcommand
docker run --rm -v $(pwd)/otel-collector.yaml:/etc/otelcol/config.yaml otel/opentelemetry-collector-contrib:latest validate --config=/etc/otelcol/config.yaml
```
If the configuration is valid, the command returns exit code 0.

---

# 🐶 Datadog SLO Verification

Datadog does not provide a local emulator or free tier (outside of their 14-day trials). 

## 1. Terraform Validation (Local)
You can verify the format against the Datadog Terraform provider schema locally:

1. Create a `main.tf` file:
```hcl
terraform {
  required_providers {
    datadog = {
      source = "DataDog/datadog"
    }
  }
}

provider "datadog" {
  api_key = "mock-key"
  app_key = "mock-key"
}

locals {
  slo = jsondecode(file("datadog-slo.json"))
}

resource "datadog_service_level_objective" "generated" {
  name        = local.slo.name
  type        = local.slo.type
  description = local.slo.description
  
  query {
    numerator   = local.slo.query.numerator
    denominator = local.slo.query.denominator
  }
  
  dynamic "thresholds" {
    for_each = local.slo.thresholds
    content {
      timeframe = thresholds.value.timeframe
      target    = thresholds.value.target
      warning   = thresholds.value.warning
    }
  }
  
  tags = local.slo.tags
}
```

2. Run validation locally:
```bash
# Generate payload
./sloak render --template datadog --slo 99.9 --window 30d --values templates/values.yaml > datadog-slo.json

# Option A: Using local Terraform CLI
terraform init
terraform validate

# Option B: Containerized validation (no local Terraform installation required)
docker run --rm -v $(pwd):/workspace -w /workspace hashicorp/terraform:latest init
docker run --rm -v $(pwd):/workspace -w /workspace hashicorp/terraform:latest validate
```

## 2. TODO for Local Testing
> [!NOTE]
> The Datadog Agent container only collects and forwards telemetry from your host system to Datadog's SaaS platform. It does not emulate or host the Datadog API. Therefore, testing the `curl` upload below requires hitting the real Datadog SaaS API endpoints with active API/Application keys.

* [ ] Verify uploading to the Datadog API directly using your actual credentials:
  ```bash
  curl -X POST "https://api.datadoghq.com/api/v1/slo" \
    -H "Accept: application/json" \
    -H "Content-type: application/json" \
    -H "DD-API-KEY: ${DD_API_KEY}" \
    -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
    -d @datadog-slo.json
  ```

