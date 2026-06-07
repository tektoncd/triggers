# ADR-0002: Interceptor Communication via HTTP(S) Webhook

**Status:** Accepted
**Date:** 2024-01-01
**TEPs:** [TEP-0026](https://github.com/tektoncd/community/blob/main/teps/0026-interceptor-plugins.md)

## Context

Triggers needs to support both built-in interceptors (CEL, GitHub, GitLab, Bitbucket, Slack)
and user-defined custom interceptors, while keeping the extension model language-agnostic.

## Decision

All interceptors — core and custom — are invoked by the sink via **HTTP(S) POST** using
a JSON `InterceptorRequest`/`InterceptorResponse` contract:

- **Core interceptors** (CEL, GitHub, GitLab, Bitbucket, Slack) run as a dedicated
  in-cluster HTTPS service (`cmd/interceptors`, port 8443) separate from the sink.
- **Custom interceptors** are deployed as separate services and registered via the
  `ClusterInterceptor` CRD; the sink calls them at the configured HTTP(S) URL.

## Architectural Invariants

- The `InterceptorRequest`/`InterceptorResponse` JSON contract over HTTP(S) is the
  canonical interface. Any interceptor — core or custom — must implement this contract.
- Interceptors are called sequentially in the order defined in the `EventListener` spec.
  A failed interceptor short-circuits the chain; subsequent interceptors are not called.
- An interceptor that returns `continue: false` stops processing without creating
  any Kubernetes resources.
- Core interceptors share a single process and must not share mutable state between
  requests.

## Preconditions

- Core interceptors service must be reachable from the sink within the cluster.
- Custom interceptors must be reachable from the EventListener sink via the address
  specified in the `ClusterInterceptor` resource.
- TLS is used by default for core interceptors (port 8443); recommended for custom
  interceptors in production.

## Consequences

- Custom interceptors can be implemented in any language with an HTTP server
- Core interceptors incur an in-cluster network hop from the sink (not in-process)
- Adding a new core interceptor requires a change to the interceptors binary
- Custom interceptor failures surface as EventListener processing errors
