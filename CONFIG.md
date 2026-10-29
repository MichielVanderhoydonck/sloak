# Configuration Rationale

SLOAK uses a robust file-based and environment-based configuration system (powered by Viper) to help teams define their standard Service Level Objectives without needing to pass the same flags on every command. 

This document explains our methodology for what is included in the `.sloak.yaml` file and what is strictly excluded.

## Included: Global Defaults

The configuration file is intended for **stable, universal settings** that apply to your team or organization as a whole.

- `slo`: Your baseline Service Level Objective percentage (e.g., `99.9`).
- `window`: Your standard calculation time window (e.g., `30d`).
- `mttr`: Your team's Mean Time To Recover baseline (e.g., `30m`). Used as a default for feasibility calculations.
- `cost`: Your estimated downtime cost per event (e.g., `10s` for a deployment or restart). Used as a default for max-disruption calculations.

*By placing these in your `~/.sloak.yaml` or setting them via environment variables like `SLOAK_SLO`, you can run commands like `sloak calculate errorbudget` without any additional arguments.*

## Excluded: Context-Specific Inputs

A configuration file should not be a dumping ground for every possible flag. We explicitly exclude flags that represent context-specific questions or volatile data.

### 1. Ephemeral Incident & Traffic State
- **`elapsed`, `consumed`, `failed-requests`, & `rps`** *(from `calculate burnrate` and `calculate errorbudget`)*: These describe the traffic rate of a specific service or the state of an ongoing incident (e.g., "we are 5 days in and consumed 2 hours or 180,000 failed requests at 100 RPS"). This changes per service and incident.

### 2. Ad-hoc Architecture Parameters
- **`components` & `type`** *(from `calculate dependency`)*: These describe the specific system architecture you are evaluating *right now* (e.g., "calculate `api=99.99,db1=99.9|db2=99.9,auth=99.9` in series"). Note that while `calculate dependency` uses `window` for expected composite downtime, its `--slo` flag is only evaluated when explicitly passed on the CLI so global `.sloak.yaml` defaults do not trigger unintended target checks during ad-hoc exploration.

### 3. Conversion Parameters
- **`nines` & `downtime`** *(from `convert`)*: These are the direct input values you want the tool to translate (e.g., "What is 99.95% in downtime?"). 

### 4. Third-Party Exporters
- **`template`, `values`, `spec`, `set`, `set-json`, `format`** *(from `render`)*: These define the output structure and configuration for third-party systems like Prometheus Operator or Datadog. Because these are specific to the output target or manifest being generated, they are provided via explicit flags rather than a global config file.

## Excluded: Universal CLI Mechanics

These flags control how the application behaves on a system level, rather than defining SRE metrics:

- **`config`**: Overrides the location of the config file itself (`--config custom.yaml`). Putting `config: custom.yaml` *inside* the config file creates a recursive paradox.
- **`help`**: A standard Cobra flag used to execute the help menu action.
- **`output`**: Controls whether output is human-readable or `json`. While technically possible to default to JSON globally, it is discouraged as it forces all casual CLI usage into rigid JSON payloads unless explicitly overridden.

---

> **Note to Developers:** If you add a new flag to SLOAK, consider if it represents a global SRE constant or a volatile input. If it is a constant, add it to the `init.go` template. If it is volatile, add it to the `ignoredFlags` map in `test/e2e/init_flags_test.go` to keep the regression tests passing!
