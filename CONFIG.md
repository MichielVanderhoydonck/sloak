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

### 1. Ephemeral Incident State
- **`elapsed` & `consumed`** *(from `calculate burnrate`)*: These describe the state of an ongoing, specific incident (e.g., "we are 5 days in and consumed 2 hours"). This changes every time you run the command.

### 2. Ad-hoc Architecture Parameters
- **`components` & `type`** *(from `calculate dependency`)*: These describe the specific system architecture you are evaluating *right now* (e.g., "calculate these three 99.9% services in parallel"). A user will evaluate many different systems throughout the day.

### 3. Conversion Parameters
- **`nines` & `downtime`** *(from `convert`)*: These are the direct input values you want the tool to translate (e.g., "What is 99.95% in downtime?"). 

### 4. Third-Party Exporters
- **`metric-name`, `rule-labels`, `namespace`** *(from `generate prometheus`)*: These define the strict output structure for third-party systems like Prometheus Operator. Because these are highly sensitive to the cluster/namespace being targeted, they are kept as explicit flags to prevent accidental cross-contamination from a global config file.

## Excluded: Universal CLI Mechanics

These flags control how the application behaves on a system level, rather than defining SRE metrics:

- **`config`**: Overrides the location of the config file itself (`--config custom.yaml`). Putting `config: custom.yaml` *inside* the config file creates a recursive paradox.
- **`help`**: A standard Cobra flag used to execute the help menu action.
- **`output`**: Controls whether output is human-readable or `json`. While technically possible to default to JSON globally, it is discouraged as it forces all casual CLI usage into rigid JSON payloads unless explicitly overridden.

---

> **Note to Developers:** If you add a new flag to SLOAK, consider if it represents a global SRE constant or a volatile input. If it is a constant, add it to the `init.go` template. If it is volatile, add it to the `ignoredFlags` map in `test/e2e/init_flags_test.go` to keep the regression tests passing!
