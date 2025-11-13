# internal/adapter

This directory holds our **Driven (Secondary) Adapters**.

In Hexagonal Architecture, these components are "driven by" our application's core.   
They provide concrete implementations for the interfaces (Ports) defined in `internal/core/port/`.

## Purpose

This folder contains all the code that talks to external infrastructure.   
It's the "edge" of our application where we interact with the "real world."

### Examples

* External API clients (e.g., Prometheus, Datadog)
* Notification services (e.g., Slack, PagerDuty)

## Rule of Thumb

**Dependencies MUST point inward.**

* **OK:** `internal/adapter` imports `internal/core/port`
* **NEVER:** `internal/core` imports `internal/adapter`