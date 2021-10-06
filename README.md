# Tekton Triggers

[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/tektoncd/triggers)
[![Go Report Card](https://goreportcard.com/badge/tektoncd/triggers)](https://goreportcard.com/report/github.com/tektoncd/triggers)


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

## Background

[Tekton](https://github.com/tektoncd/pipeline) is a Kubernetes-native continuous integration and delivery
(CI/CD) framework that allows you to create containerized, composable, and configurable workloads declaratively
through Kubernetes CRDs. When integrated with Tekton Triggers, Tekton allows you to easily create fully fledged CI/CD systems in which you
define all mechanics exclusively using Kubernetes resources.

To learn more, see the [Tekton Triggers Overview](docs/README.md).

## Getting Started

To get started with Tekton Triggers, see the following:

* [Overview of Tekton Triggers](./docs/README.md)
* [Setting Up Tekton Triggers](./docs/install.md)
* [Getting Started with Tekton Triggers](./docs/getting-started/README.md)
* [Tekton Triggers code examples](./examples/README.md)

The "Getting Started with Tekton Triggers" guide walks you through setting up an end-to-end image building solution triggered via GitHub's `push` events.

### Documentation

| Version                                                                                  | Docs                                                                                   | Examples                                                                                | Getting Started                                                                                                                 |
| ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| [HEAD](https://github.com/tektoncd/triggers/blob/main/DEVELOPMENT.md#install-pipeline) | [Docs @ HEAD](https://github.com/tektoncd/triggers/blob/main/docs/README.md)         | [Examples @ HEAD](https://github.com/tektoncd/triggers/blob/main/examples)            | [Getting Started @ HEAD](https://github.com/tektoncd/triggers/blob/main/docs/getting-started#getting-started-with-triggers)   |
| [v0.16.0](https://github.com/tektoncd/triggers/releases/tag/v0.16.0)                       | [Docs @ v0.16.0](https://github.com/tektoncd/triggers/tree/v0.16.0/docs#tekton-triggers) | [Examples @ v0.16.0](https://github.com/tektoncd/triggers/tree/v0.16.0/examples#examples) | [Getting Started @ v0.16.0](https://github.com/tektoncd/triggers/tree/v0.16.0/docs/getting-started#getting-started-with-triggers) |
| [v0.15.2](https://github.com/tektoncd/triggers/releases/tag/v0.15.2)                       | [Docs @ v0.15.2](https://github.com/tektoncd/triggers/tree/v0.15.2/docs#tekton-triggers) | [Examples @ v0.15.2](https://github.com/tektoncd/triggers/tree/v0.15.2/examples#examples) | [Getting Started @ v0.15.2](https://github.com/tektoncd/triggers/tree/v0.15.2/docs/getting-started#getting-started-with-triggers) |
| [v0.15.1](https://github.com/tektoncd/triggers/releases/tag/v0.15.1)                       | [Docs @ v0.15.1](https://github.com/tektoncd/triggers/tree/v0.15.1/docs#tekton-triggers) | [Examples @ v0.15.1](https://github.com/tektoncd/triggers/tree/v0.15.1/examples#examples) | [Getting Started @ v0.15.1](https://github.com/tektoncd/triggers/tree/v0.15.1/docs/getting-started#getting-started-with-triggers) |
| [v0.15.0](https://github.com/tektoncd/triggers/releases/tag/v0.15.0)                       | [Docs @ v0.15.0](https://github.com/tektoncd/triggers/tree/v0.15.0/docs#tekton-triggers) | [Examples @ v0.15.0](https://github.com/tektoncd/triggers/tree/v0.15.0/examples#examples) | [Getting Started @ v0.15.0](https://github.com/tektoncd/triggers/tree/v0.15.0/docs/getting-started#getting-started-with-triggers) |
| [v0.14.2](https://github.com/tektoncd/triggers/releases/tag/v0.14.2)                       | [Docs @ v0.14.2](https://github.com/tektoncd/triggers/tree/v0.14.2/docs#tekton-triggers) | [Examples @ v0.14.2](https://github.com/tektoncd/triggers/tree/v0.14.2/examples#examples) | [Getting Started @ v0.14.2](https://github.com/tektoncd/triggers/tree/v0.14.2/docs/getting-started#getting-started-with-triggers) |
| [v0.14.0](https://github.com/tektoncd/triggers/releases/tag/v0.14.0)                       | [Docs @ v0.14.0](https://github.com/tektoncd/triggers/tree/v0.14.0/docs#tekton-triggers) | [Examples @ v0.14.0](https://github.com/tektoncd/triggers/tree/v0.14.0/examples#examples) | [Getting Started @ v0.14.0](https://github.com/tektoncd/triggers/tree/v0.14.0/docs/getting-started#getting-started-with-triggers) |
| [v0.13.0](https://github.com/tektoncd/triggers/releases/tag/v0.13.0)                       | [Docs @ v0.13.0](https://github.com/tektoncd/triggers/tree/v0.13.0/docs#tekton-triggers) | [Examples @ v0.13.0](https://github.com/tektoncd/triggers/tree/v0.13.0/examples#examples) | [Getting Started @ v0.13.0](https://github.com/tektoncd/triggers/tree/v0.13.0/docs/getting-started#getting-started-with-triggers) |
| [v0.12.1](https://github.com/tektoncd/triggers/releases/tag/v0.12.1)                       | [Docs @ v0.12.1](https://github.com/tektoncd/triggers/tree/v0.12.1/docs#tekton-triggers) | [Examples @ v0.12.1](https://github.com/tektoncd/triggers/tree/v0.12.1/examples#examples) | [Getting Started @ v0.12.1](https://github.com/tektoncd/triggers/tree/v0.12.1/docs/getting-started#getting-started-with-triggers) |
| [v0.12.0](https://github.com/tektoncd/triggers/releases/tag/v0.12.0)                       | [Docs @ v0.12.0](https://github.com/tektoncd/triggers/tree/v0.12.0/docs#tekton-triggers) | [Examples @ v0.12.0](https://github.com/tektoncd/triggers/tree/v0.12.0/examples#examples) | [Getting Started @ v0.12.0](https://github.com/tektoncd/triggers/tree/v0.12.0/docs/getting-started#getting-started-with-triggers) |
| [v0.11.2](https://github.com/tektoncd/triggers/releases/tag/v0.11.2)                       | [Docs @ v0.11.2](https://github.com/tektoncd/triggers/tree/v0.11.2/docs#tekton-triggers) | [Examples @ v0.11.2](https://github.com/tektoncd/triggers/tree/v0.11.2/examples#examples) | [Getting Started @ v0.11.2](https://github.com/tektoncd/triggers/tree/v0.11.2/docs/getting-started#getting-started-with-triggers) |
| [v0.11.1](https://github.com/tektoncd/triggers/releases/tag/v0.11.1)                       | [Docs @ v0.11.1](https://github.com/tektoncd/triggers/tree/v0.11.1/docs#tekton-triggers) | [Examples @ v0.11.1](https://github.com/tektoncd/triggers/tree/v0.11.1/examples#examples) | [Getting Started @ v0.11.1](https://github.com/tektoncd/triggers/tree/v0.11.1/docs/getting-started#getting-started-with-triggers) |
| [v0.11.0](https://github.com/tektoncd/triggers/releases/tag/v0.11.0)                       | [Docs @ v0.11.0](https://github.com/tektoncd/triggers/tree/v0.11.0/docs#tekton-triggers) | [Examples @ v0.11.0](https://github.com/tektoncd/triggers/tree/v0.11.0/examples#examples) | [Getting Started @ v0.11.0](https://github.com/tektoncd/triggers/tree/v0.11.0/docs/getting-started#getting-started-with-triggers) |
| [v0.10.2](https://github.com/tektoncd/triggers/releases/tag/v0.10.2)                       | [Docs @ v0.10.2](https://github.com/tektoncd/triggers/tree/v0.10.2/docs#tekton-triggers) | [Examples @ v0.10.2](https://github.com/tektoncd/triggers/tree/v0.10.2/examples#examples) | [Getting Started @ v0.10.2](https://github.com/tektoncd/triggers/tree/v0.10.2/docs/getting-started#getting-started-with-triggers) |
| [v0.10.1](https://github.com/tektoncd/triggers/releases/tag/v0.10.1)                       | [Docs @ v0.10.1](https://github.com/tektoncd/triggers/tree/v0.10.1/docs#tekton-triggers) | [Examples @ v0.10.1](https://github.com/tektoncd/triggers/tree/v0.10.1/examples#examples) | [Getting Started @ v0.10.1](https://github.com/tektoncd/triggers/tree/v0.10.1/docs/getting-started#getting-started-with-triggers) |
| [v0.10.0](https://github.com/tektoncd/triggers/releases/tag/v0.10.0)                       | [Docs @ v0.10.0](https://github.com/tektoncd/triggers/tree/v0.10.0/docs#tekton-triggers) | [Examples @ v0.10.0](https://github.com/tektoncd/triggers/tree/v0.10.0/examples#examples) | [Getting Started @ v0.10.0](https://github.com/tektoncd/triggers/tree/v0.10.0/docs/getting-started#getting-started-with-triggers) |
| [v0.9.1](https://github.com/tektoncd/triggers/releases/tag/v0.9.1)                       | [Docs @ v0.9.1](https://github.com/tektoncd/triggers/tree/v0.9.1/docs#tekton-triggers) | [Examples @ v0.9.1](https://github.com/tektoncd/triggers/tree/v0.9.1/examples#examples) | [Getting Started @ v0.9.1](https://github.com/tektoncd/triggers/tree/v0.9.1/docs/getting-started#getting-started-with-triggers) |
| [v0.9.0](https://github.com/tektoncd/triggers/releases/tag/v0.9.0)                       | [Docs @ v0.9.0](https://github.com/tektoncd/triggers/tree/v0.9.0/docs#tekton-triggers) | [Examples @ v0.9.0](https://github.com/tektoncd/triggers/tree/v0.9.0/examples#examples) | [Getting Started @ v0.9.0](https://github.com/tektoncd/triggers/tree/v0.9.0/docs/getting-started#getting-started-with-triggers) |
| [v0.8.1](https://github.com/tektoncd/triggers/releases/tag/v0.8.1)                       | [Docs @ v0.8.1](https://github.com/tektoncd/triggers/tree/v0.8.1/docs#tekton-triggers) | [Examples @ v0.8.1](https://github.com/tektoncd/triggers/tree/v0.8.1/examples#examples) | [Getting Started @ v0.8.1](https://github.com/tektoncd/triggers/tree/v0.8.1/docs/getting-started#getting-started-with-triggers) |
| [v0.8.0](https://github.com/tektoncd/triggers/releases/tag/v0.8.0)                       | [Docs @ v0.8.0](https://github.com/tektoncd/triggers/tree/v0.8.0/docs#tekton-triggers) | [Examples @ v0.8.0](https://github.com/tektoncd/triggers/tree/v0.8.0/examples#examples) | [Getting Started @ v0.8.0](https://github.com/tektoncd/triggers/tree/v0.8.0/docs/getting-started#getting-started-with-triggers) |
| [v0.7.0](https://github.com/tektoncd/triggers/releases/tag/v0.7.0)                       | [Docs @ v0.7.0](https://github.com/tektoncd/triggers/tree/v0.7.0/docs#tekton-triggers) | [Examples @ v0.7.0](https://github.com/tektoncd/triggers/tree/v0.7.0/examples#examples) | [Getting Started @ v0.7.0](https://github.com/tektoncd/triggers/tree/v0.7.0/docs/getting-started#getting-started-with-triggers) |
| [v0.6.1](https://github.com/tektoncd/triggers/releases/tag/v0.6.1)                       | [Docs @ v0.6.1](https://github.com/tektoncd/triggers/tree/v0.6.1/docs#tekton-triggers) | [Examples @ v0.6.1](https://github.com/tektoncd/triggers/tree/v0.6.1/examples#examples) | [Getting Started @ v0.6.1](https://github.com/tektoncd/triggers/tree/v0.6.1/docs/getting-started#getting-started-with-triggers) |
| [v0.6.0](https://github.com/tektoncd/triggers/releases/tag/v0.6.0)                       | [Docs @ v0.6.0](https://github.com/tektoncd/triggers/tree/v0.6.0/docs#tekton-triggers) | [Examples @ v0.6.0](https://github.com/tektoncd/triggers/tree/v0.6.0/examples#examples) | [Getting Started @ v0.6.0](https://github.com/tektoncd/triggers/tree/v0.6.0/docs/getting-started#getting-started-with-triggers) |
| [v0.5.0](https://github.com/tektoncd/triggers/releases/tag/v0.5.0)                       | [Docs @ v0.5.0](https://github.com/tektoncd/triggers/tree/v0.5.0/docs#tekton-triggers) | [Examples @ v0.5.0](https://github.com/tektoncd/triggers/tree/v0.5.0/examples#examples) | [Getting Started @ v0.5.0](https://github.com/tektoncd/triggers/tree/v0.5.0/docs/getting-started#getting-started-with-triggers) |
| [v0.4.0](https://github.com/tektoncd/triggers/releases/tag/v0.4.0)                       | [Docs @ v0.4.0](https://github.com/tektoncd/triggers/tree/v0.4.0/docs#tekton-triggers) | [Examples @ v0.4.0](https://github.com/tektoncd/triggers/tree/v0.4.0/examples#examples) | [Getting Started @ v0.4.0](https://github.com/tektoncd/triggers/tree/v0.4.0/docs/getting-started#getting-started-with-triggers) |
| [v0.3.1](https://github.com/tektoncd/triggers/releases/tag/v0.3.1)                       | [Docs @ v0.3.1](https://github.com/tektoncd/triggers/tree/v0.3.1/docs#tekton-triggers) | [Examples @ v0.3.1](https://github.com/tektoncd/triggers/tree/v0.3.1/examples#examples) | [Getting Started @ v0.3.1](https://github.com/tektoncd/triggers/tree/v0.3.1/docs/getting-started#getting-started-with-triggers) |
| [v0.3.0](https://github.com/tektoncd/triggers/releases/tag/v0.3.0)                       | [Docs @ v0.3.0](https://github.com/tektoncd/triggers/tree/v0.3.0/docs#tekton-triggers) | [Examples @ v0.3.0](https://github.com/tektoncd/triggers/tree/v0.3.0/examples#examples) | [Getting Started @ v0.3.0](https://github.com/tektoncd/triggers/tree/v0.3.0/docs/getting-started#getting-started-with-triggers) |
| [v0.2.1](https://github.com/tektoncd/triggers/releases/tag/v0.2.1)                       | [Docs @ v0.2.1](https://github.com/tektoncd/triggers/tree/v0.2.1/docs#tekton-triggers) | [Examples @ v0.2.1](https://github.com/tektoncd/triggers/tree/v0.2.1/examples#examples) | [Getting Started @ v0.2.1](https://github.com/tektoncd/triggers/tree/v0.2.1/docs/getting-started#getting-started-with-triggers) |
| [v0.1.0](https://github.com/tektoncd/triggers/releases/tag/v0.1.0)                       | [Docs @ v0.1.0](https://github.com/tektoncd/triggers/tree/v0.1.0/docs#tekton-triggers) | [Examples @ v0.1.0](https://github.com/tektoncd/triggers/tree/v0.1.0/examples#examples) | [Getting Started @ v0.1.0](https://github.com/tektoncd/triggers/tree/v0.1.0/docs/getting-started#getting-started-with-triggers) |

## Want to contribute?

Hooray!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes.
- See [DEVELOPMENT.md](DEVELOPMENT.md) to get started.
- Look at our [good first issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our [help wanted issues](https://github.com/tektoncd/triggers/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) to help improve Tekton Triggers.
