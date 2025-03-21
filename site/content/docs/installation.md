---
title: "Installation"
weight: 10
# bookFlatSection: false
# bookToc: true
# bookHidden: false
# bookCollapseSection: false
# bookComments: false
# bookSearchExclude: false
---
# Installation

## Installing from Source
To install quad-ops from source, you'll need to have Go installed on your system. Once you have Go installed, you can clone the quad-ops repository and build the binary:
```bash
git clone https://github.com/trly/quad-ops.git
cd quad-ops
go build
```
The binary will be built in the current directory. You can then move it to a directory in your PATH, such as /usr/local/bin, and make it executable:
```bash
sudo mv quad-ops /usr/local/bin/quad-ops
sudo chmod +x /usr/local/bin/quad-ops
```
## Installing a Pre-Built Binary
You can also download a pre-built binary for your platform from the [releases page](https://github.com/trly/quad-ops/releases). Simply download the appropriate binary for your system and make it executable.
