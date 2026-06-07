# Architecture Decision Records

Architectural decisions for Tekton Triggers are tracked as **Tekton Enhancement Proposals (TEPs)**
in the [tektoncd/community](https://github.com/tektoncd/community/tree/main/teps) repository.

TEPs follow a lightweight RFC process and cover cross-cutting concerns across all Tekton projects.
Triggers-specific TEPs are tagged accordingly in the community repo.

## Core Design Decisions

Key architectural decisions captured locally:

- [ADR-0001: Event-Driven Architecture and Component Responsibilities](./0001-event-driven-architecture.md)
- [ADR-0002: Interceptor Communication via HTTP(S) Webhook](./0002-interceptor-communication.md)

## How to Propose Changes

Open a TEP in [tektoncd/community](https://github.com/tektoncd/community/blob/main/process/tep-process.md)
following the TEP process for any change that affects the public API, CRD schema, or cross-component behavior.
