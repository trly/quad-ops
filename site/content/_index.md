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
![Docs Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml)
![Build Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/docs.yaml?label=docs)
![CodeQL Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/trly/quad-ops/build.yml?label=codeql)
![GitHub Release](https://img.shields.io/github/v/release/trly/quad-ops)

A lightweight GitOps framework for podman containers managed by [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

Quad-Ops is a tool that helps you manage container deployments using Podman and systemd in a GitOps workflow. It watches Git repositories for container definitions written in YAML and automatically converts them into unit files that systemd can use to run your containers.

### Key Features:
- Monitor multiple Git repositories for container configurations
- Supports containers, volumes, networks and images
- Works in both system-wide and user (rootless) modes
