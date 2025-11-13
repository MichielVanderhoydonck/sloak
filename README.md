# SLOAK

## Service-Level-Objective-Army-Knife
The primary goal of this tool is to provide a set of command-line utilities for designing service-level objectives (SLOs). 
It aims to simplify the process of setting up, configuring, and monitoring SLOs, making it easier for developers and operations teams to ensure high availability and performance.
More information on implementing SLOs can be found on the [Google SRE book site](https://sre.google/workbook/implementing-slos/)

## Word of Warning
This is work in progress and should not be considered for actual use yet.

## Architecture
The main idea is to use a [clean architecture](https://threedots.tech/post/introducing-clean-architecture/) while staying compliant to framework guidelines.
Current main frameworks used is [Cobra](https://github.com/spf13/cobra).  
Each subfolder of the `cmd` folder should be self contained, only exposing the main command via a `NewCommand` function.

The `main.go` file should be responsible for initializing the application and running the main command exposed through the root command containing the Cobra bootstrapping.

### go-arch-lint
This repo includes config for linting architecture using [go-arch-lint](https://github.com/uber-go/go-arch-lint).  
Note that this currently errros since that linter is still pinned on 1.23. (downgrade the mod file for now)
Simply run:
```bash
docker run --rm -v ${PWD}:/app fe3dback/go-arch-lint:latest-stable-release check --project-path /app
```

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

