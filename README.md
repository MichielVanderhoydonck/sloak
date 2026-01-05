<img src="./sloak.png" width="256"> 

## Service-Level-Objective-Army-Knife
The primary goal of this tool is to provide a set of command-line utilities for designing service-level objectives (SLOs). 
It aims to simplify the process of setting up, configuring, and monitoring SLOs, making it easier for developers and operations teams to ensure high availability and performance.
More information on implementing SLOs can be found on the [Google SRE book site](https://sre.google/workbook/implementing-slos/)

## Architecture
The main idea is to use a [clean architecture](https://threedots.tech/post/introducing-clean-architecture/) while staying compliant to framework guidelines.
Current main frameworks used is [Cobra](https://github.com/spf13/cobra).  
Each subfolder of the `cmd` folder should be self contained, only exposing the main command via a `NewCommand` function.

The `main.go` file should be responsible for initializing the application and running the main command exposed through the root command containing the Cobra bootstrapping.

### go-arch-lint
This repo includes config for linting architecture using [go-arch-lint](https://github.com/fe3dback/go-arch-lint).  
Simply run:
```bash
docker run --rm -v ${PWD}:/app fe3dback/go-arch-lint:latest-stable-release check --project-path /app
```

## Commands
Here follows a quick reference table of available commands, use the `--help` flag to get more information.
| command | sub-command | info |
|---------|-------------|------|
| **sloak calculate** || main calculate root |
|| **burnrate** | Calculates the current error budget burn rate (consumption speed).|
|| **dependency** | Calculates composite availability for serial or parallel dependencies.|
|| **errorbudget** | Calculates the total error budget (time) for a given SLO.|
|| **feasibility** |Calculates if an SLO is realistic given your MTTR.
|| **max-disruption** | Calculates allowed deployment frequency based on disruption cost.|
|| **translator** | Translates between Availability % and Downtime Duration.|
|**sloak generate**||main generate root|
||**alert-table** | Generates the standard SRE Alerting Table.|

## Testing

Domain and Service Logic can be tested via:
```bash
go test ./...
```

Adapter tests (Mock based) test our driving adapter (Cobra CLI).
Testing a specific command can be done by:
```bash
go test ./cmd/burnrate
```

Binary testing can be done invocing the following:
```bash
go test -v -tags=e2e ./test/e2e
```

## Building

Creating a binary to run is as simple as:
```bash
go build -o sloak cmd/sloak/main.go
```
