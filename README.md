# Tekton Triggers

[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/tektoncd/triggers)
[![Go Report Card](https://goreportcard.com/badge/tektoncd/triggers)](https://goreportcard.com/report/github.com/tektoncd/triggers)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/6527/badge)](https://bestpractices.coreinfrastructure.org/projects/6527)


<p align="center">
<img src="tekton-triggers.png" alt="Tekton Triggers logo (Tekton cat playing with a ball)"></img>
</p>

Tekton Triggers is a Kubernetes
[Custom Resource Definition](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
(CRD) controller that allows you to create Kubernetes resources based on information it extracts from event payloads.

 Tekton Triggers originates from the implementation of [this design](https://docs.google.com/document/d/1fngeNn3kGD4P_FTZjAnfERcEajS7zQhSEUaN7BYIlTw/edit#heading=h.iyqzt1brkg3o)
(visible to members of [the Tekton mailing list](https://github.com/tektoncd/community/blob/main/contact.md#mailing-list)).

* [Background](#background)
* [Getting Started](#getting-started)
* [Want to contribute?](#want-to-contribute)
* [Project roadmap](roadmap.md)
* Discover our [releases](releases.md)

## Background

[Tekton](https://github.com/tektoncd/pipeline) is a Kubernetes-native continuous integration and delivery
(CI/CD) framework that allows you to create containerized, composable, and configurable workloads declaratively
through Kubernetes CRDs. When integrated with Tekton Triggers, Tekton allows you to easily create fully fledged CI/CD systems in which you
define all mechanics exclusively using Kubernetes resources.

To learn more, see the [Tekton Triggers Overview](docs/README.md).

## Getting Started

To get started with Tekton Triggers, see the latest version of our docs:

* [Overview of Tekton Triggers](./docs/README.md)
* [Setting Up Tekton Triggers](./docs/install.md)
* [Getting Started with Tekton Triggers](./docs/getting-started/README.md)
* [Tekton Triggers code examples](./examples/README.md)

The "Getting Started with Tekton Triggers" guide walks you through setting up an end-to-end image building solution triggered via GitHub's `push` events.

Version specific links are available in the [releases](releases.md) page and on the
[Tekton website](https://tekton.dev/docs).

## Want to contribute?

Hooray!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes.
- See [DEVELOPMENT.md](DEVELOPMENT.md) to get started.
- Look at our [good first issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our [help wanted issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to help improve Tekton Triggers.
