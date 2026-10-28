# internal/adapter

This directory holds our **Adapters** for external specifications, rendering engines, and vendor SDKs.

In Hexagonal / Clean Architecture, adapters implement domain ports to translate between external representations/engines and pure domain models.

## Adapters

* **`openslo`**: Reads and validates external OpenSLO YAML specifications (using the official OpenSLO Go SDK) and maps them into domain `SLOSpec` models (`domain.SpecLoader`).
* **`cue`**: Evaluates CUE templates against the embedded schema, validates PromQL expressions via the Prometheus AST parser, and serializes output to YAML/JSON (`domain.RenderEngine`).

## Rule of Thumb

* **Dependencies MUST point inward.**
* `internal/adapter` may depend on `internal/domain`, `internal/util`, `templates`, and external vendor SDKs.
* `internal/domain` and `internal/service` MUST NOT depend on `internal/adapter` or external vendor SDKs (pure Go, zero external dependencies).
