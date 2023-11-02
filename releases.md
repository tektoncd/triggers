# Tekton Triggers Releases

## Release Frequency

Tekton Triggers follows the Tekton community [release policy][release-policy]
as follows:

- Versions are numbered according to semantic versioning: `vX.Y.Z`
- A new release is produced on a monthly basis, when enough content is available
- Four releases a year are chosen for [long term support (LTS)](https://github.com/tektoncd/community/blob/main/releases.md#support-policy).
  All remaining releases are supported for approximately 1 month (until the next
  release is produced)
    - LTS releases take place in January, April, July and October every year
    - The first Tekton Triggers LTS release will be **v0.22.0** in October 2022
    - Releases usually happen towards the middle of the month, but the exact date
      may vary, depending on week-ends and readiness

Tekton Triggers produces nightly builds, publicly available on
`gcr.io/tekton-nightly`. 

### Transition Process

Before release v0.22 Tekton Triggers has worked on the basis of an undocumented
support period of four months, which will be maintained for the release v0.21.

## Release Process

Tekton Triggers releases are made of YAML manifests and container images.
Manifests are published to cloud object-storage as well as
[GitHub][tekton-triggers-releases]. Container images are signed by
[Sigstore][sigstore] via [Tekton Chains][tekton-chains]; signatures can be
verified through the [public key][chains-public-key] hosted by the Tekton Chains
project.

Further documentation available:

- The Tekton Triggers [release process][tekton-releases-docs]
- [Installing Tekton][tekton-installation]
- Standard for [release notes][release-notes-standards]

## Releases

### v0.25

- **Latest Release**: [v0.25.2][v0-25-2] (2023-10-20) ([docs][v0-25-2-docs], [examples][v0-25-2-examples])
- **Initial Release**: [v0.25.0][v0-25-0] (2023-08-29)
- **End of Life**: 2023-11-11
- **Patch Releases**: [v0.25.0][v0-25-0] [v0.25.1][v0-25-1]

### v0.24 (LTS)

- **Latest Release**: [v0.24.1][v0-24-1] (2023-06-11) ([docs][v0-24-1-docs], [examples][v0-24-1-examples])
- **Initial Release**: [v0.24.0][v0-24-0] (2023-06-11)
- **End of Life**: 2024-06-11
- **Patch Releases**: [v0.24.1][v0-24-1] [v0.24.0][v0-24-0]

### Required Kubernetes Version

- Starting from the v0.23.x release of Triggers: **Kubernetes version 1.24 or later**
- Starting from the v0.22.x release of Triggers: **Kubernetes version 1.23 or later**
- Starting from the v0.21.x release of Triggers: **Kubernetes version 1.22 or later**

## End of Life Releases

Older releases are EOL and available on [GitHub][tekton-triggers-releases].


[release-policy]: https://github.com/tektoncd/community/blob/main/releases.md
[sigstore]: https://sigstore.dev
[tekton-chains]: https://github.com/tektoncd/chains
[tekton-triggers-releases]: https://github.com/tektoncd/triggers/releases
[chains-public-key]: https://github.com/tektoncd/chains/blob/main/tekton.pub
[tekton-releases-docs]: tekton/README.md
[tekton-installation]: docs/install.md
[release-notes-standards]:
    https://github.com/tektoncd/community/blob/main/standards.md#release-notes

[v0-24-0]: https://github.com/tektoncd/triggers/releases/tag/v0.24.0
[v0-24-1]: https://github.com/tektoncd/triggers/releases/tag/v0.24.1
[v0-24-2]: https://github.com/tektoncd/triggers/releases/tag/v0.24.2
[v0-24-2-docs]: https://github.com/tektoncd/triggers/tree/v0.24.2/docs#tekton-triggers
[v0-24-2-examples]: https://github.com/tektoncd/triggers/tree/v0.24.2/examples#examples
[v0-25-0]: https://github.com/tektoncd/triggers/releases/tag/v0.25.0
[v0-25-1]: https://github.com/tektoncd/triggers/releases/tag/v0.25.1
[v0-25-2]: https://github.com/tektoncd/triggers/releases/tag/v0.25.2
[v0-25-2-docs]: https://github.com/tektoncd/triggers/tree/v0.25.2/docs#tekton-triggers
[v0-25-2-examples]: https://github.com/tektoncd/triggers/tree/v0.25.2/examples#examples

