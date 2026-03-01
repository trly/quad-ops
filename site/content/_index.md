---
title: "quad-ops"
weight: 0
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---

# ![quad-ops](images/quad-ops-64.png) quad-ops

## GitOps for Quadlet
![GitHub License](https://img.shields.io/github/license/trly/quad-ops)
![Build Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml)
![Docs Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/docs.yaml?label=docs)
![CodeQL Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/codeql.yml?label=codeql)
![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops)
[![codecov](https://codecov.io/gh/trly/quad-ops/graph/badge.svg?token=ID6CGJPXR6)](https://codecov.io/gh/trly/quad-ops)

A cross-platform GitOps framework for container management with native service integration

Quad-Ops is a tool that helps you manage container deployments using a GitOps workflow.
It watches Git repositories for standard [Docker Compose](https://compose-spec.io/) files and automatically converts them into native service definitions for your platform:

- **Linux**: systemd + [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- **macOS**: launchd (planned)

## Docker Compose Feature Support

Quad-Ops supports a wide range of Docker Compose features and converts them into Podman Quadlet directives.
See the [Compose Feature Support]({{< relref "/docs/compose-support" >}}) page for a complete annotated reference
of supported and unsupported features, including all `x-quad-ops-*` extensions.
