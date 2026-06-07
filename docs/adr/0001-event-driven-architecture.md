# ADR-0001: Event-Driven Architecture and Component Responsibilities

**Status:** Accepted
**Date:** 2024-01-01

## Context

Tekton Triggers needs to bridge external events (webhooks, CloudEvents) to Tekton
PipelineRun/TaskRun creation in a Kubernetes-native way.

## Decision

Triggers is structured around four distinct responsibilities with strict separation:

1. **EventListener** — receives and routes incoming HTTP events
2. **Interceptors** — validate and transform event payloads before trigger evaluation
3. **TriggerBinding** — extracts parameters from event payloads
4. **TriggerTemplate** — defines the Kubernetes resources to create

## Architectural Invariants

- **Only the EventListener sink creates Kubernetes resources.** Interceptors, bindings,
  and templates are pure transformation steps with no side effects.
- **Interceptors are stateless.** Each interceptor processes a request independently.
  No interceptor stores state between requests.
- **TriggerBindings are read-only.** They extract data from the event payload but never
  mutate it.
- **Tekton Pipelines must be installed before Triggers.** Triggers depends on the
  Pipeline CRDs (PipelineRun, TaskRun) being present in the cluster.
- **EventListener is the only network ingress point.** All external events enter the
  system exclusively through the EventListener HTTP endpoint.

## Preconditions

- A Kubernetes cluster with RBAC enabled
- Tekton Pipelines installed and healthy
- For TLS: valid certificates configured on the EventListener

## Consequences

- Interceptors can be developed and tested independently without a cluster
- The sink is the single point of Kubernetes API interaction, simplifying RBAC requirements
- Core and custom interceptors are both called via HTTP(S) from the sink; core interceptors
  run as a separate in-cluster service (not in-process with the sink)
- Custom interceptors can be implemented in any language that can serve HTTP(S)
