# Tekton Triggers

Event-driven CI/CD controller for Kubernetes. Listens for events (GitHub webhooks, etc.),
maps them to Tekton PipelineRuns/TaskRuns via TriggerBindings and TriggerTemplates, and
creates Kubernetes resources through EventListeners.

**Behavioral guidelines**: See [.agents/guidelines.md](./.agents/guidelines.md) for generic coding principles.

---

## Build & Test Commands

```bash
# Build all binaries
make all

# Build specific component
make bin/controller
make bin/eventlistenersink
make bin/interceptors
make bin/webhook
make bin/tkn-triggers

# Unit tests — no cluster required
make test-unit

# End-to-end tests — requires a running cluster
make test-e2e

# YAML tests
make test-yamls

# Format code — required before PR submission
make fmt

# Lint — must pass before every PR
make golangci-lint

# Code generation — required after modifying types or CRD schemas
./hack/update-codegen.sh
```

---

## Single-File Verification

```bash
# Lint a single Go file
golangci-lint run path/to/file.go

# Type-check using staticcheck
staticcheck ./path/to/pkg/...

# Format a single file in place
gofmt -w path/to/file.go

# Vet the package containing the file
go vet ./path/to/pkg/...
```

---

## Key Conventions

1. **Dependencies are vendored.** All Go dependencies live in `vendor/`. Review agents
   should ignore the `vendor/` directory — it contains third-party code.

2. **Use structured logging.** Import `knative.dev/pkg/logging` and use context-aware
   loggers. Never use `fmt.Printf` or `log.Print` in production code.

3. **CRD types live in `pkg/apis/`.** After modifying any type definition, run
   `./hack/update-codegen.sh` to regenerate deepcopy, client, and informer code.

4. **Interceptors are the request/response processing layer.** Core interceptors
   (CEL, GitHub, GitLab, Bitbucket, Slack) live in `pkg/interceptors/` and run as a
   dedicated HTTPS service (port 8443). Custom interceptors are registered via the
   `ClusterInterceptor` CRD and called by the sink over HTTP(S). The `pkg/interceptors/webhook/`
   package handles the deprecated `Interceptor.Webhook` field for calling arbitrary
   external URLs directly from the sink — distinct from `ClusterInterceptor`.

5. **Test coverage is enforced.** PRs adding functionality must include tests.
   E2E tests are tagged and require a cluster — unit tests should not.

6. **Config lives in `config/`.** Kubernetes manifests (CRDs, RBAC, deployments)
   are generated/managed there. Do not hand-edit generated CRD YAML.

---

## Architecture

**Controller** (`cmd/controller`, `pkg/reconciler/`): Reconciles EventListener,
TriggerBinding, TriggerTemplate, ClusterTriggerBinding, ClusterInterceptor CRDs.
Uses Knative's controller framework.

**EventListener Sink** (`cmd/eventlistenersink`, `pkg/sink/`): HTTP server
that receives events and processes them through trigger bindings, templates, and
interceptors to create Kubernetes resources.

**Interceptors** (`cmd/interceptors`, `pkg/interceptors/`): HTTPS service (port 8443)
handling core interceptors (CEL, GitHub, GitLab, Bitbucket, Slack). The sink calls interceptors
via HTTP POST. ClusterInterceptors can extend this for custom logic.

**Webhook** (`cmd/webhook`): Kubernetes admission webhook for validating and
defaulting CRD resources.

**tkn-triggers CLI** (`cmd/tkn-triggers`): CLI plugin for `tkn` to interact
with Triggers resources.

**Utility CLIs** (`cmd/binding-eval`, `cmd/cel-eval`, `cmd/triggerrun`): Developer
tools for evaluating bindings, CEL expressions, and trigger processing locally.

---

## Pattern References for Common Changes

- **New interceptor type**: Follow `pkg/interceptors/cel/` as the reference implementation
- **New reconciler**: Follow `pkg/reconciler/eventlistener/` for structure and patterns
- **New CRD type**: Add to `pkg/apis/triggers/`, update `./hack/update-codegen.sh`
- **Sink event processing**: See `pkg/sink/sink.go` for the main event handling path
- **E2E tests**: Follow examples in `test/eventlistener_test.go`
- **CEL expressions**: See `pkg/interceptors/cel/` and `docs/cel_expressions.md`

---

## PR Conventions

- Pull requests must follow the repository PR template in `.github/pull_request_template.md`.
- Run `make fmt` before submitting for review.
- `make golangci-lint` must pass with zero issues.
- Tests required for any functionality changes.
- Follow [Tekton commit message standards](https://github.com/tektoncd/community/blob/main/standards.md#commits).
- Add `/kind <type>` label (bug, feature, cleanup, etc.).
- Update release notes block if user-facing changes.
- Run `./hack/update-codegen.sh` after modifying CRD types.
- Ignore `vendor/` directory in reviews — contains vendored dependencies only.

---

## Skills

None configured yet. Repo-local skills can be added to `.agents/skills/`.
