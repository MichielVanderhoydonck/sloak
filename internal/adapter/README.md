# internal/adapter

This directory holds our **Adapters** for external specifications and systems.

In Hexagonal / Clean Architecture, adapters translate external representations into domain models.

## OpenSLO Adapter

The `openslo` adapter reads and validates external OpenSLO YAML specifications (using the official OpenSLO Go SDK) and maps them into domain `SLOSpec` models.

## Rule of Thumb

* **Dependencies MUST point inward.**
* `internal/adapter` may depend on `internal/domain` and external vendor SDKs.
* `internal/domain` MUST NOT depend on `internal/adapter` or external vendor SDKs (pure Go, zero external dependencies).