# Packer Plugin KubeVirt

> **Fork notice:** This repository is a fork of [hashicorp/packer-plugin-kubevirt](https://github.com/hashicorp/packer-plugin-kubevirt), maintained at [github.com/flippyboy/packer-plugin-kubevirt](https://github.com/flippyboy/packer-plugin-kubevirt).

The `KubeVirt` plugin can be used with [HashiCorp Packer](https://www.packer.io) to create KubeVirt images.

**Note**: This plugin is under development and is not production ready.

## Packer

[Packer](https://developer.hashicorp.com/packer) is a tool for creating identical machine images from a single source template.

To get started, see the [Packer installation guide](https://developer.hashicorp.com/packer/install).

## Prerequisites

- [Packer](https://packer.io)
- [Kubernetes](https://kubernetes.io) with [KubeVirt](https://kubevirt.io) installed

## Plugin Features

- **HCL Templating** – Use HashiCorp Configuration Language (HCL2) for defining infrastructure as code.
- **ISO Installation** – Build VM golden images from ISO using the `kubevirt-iso` builder.
- **ISO Media Files** – Embed additional files into installation process (e.g. `ks.cfg` or `unattend.xml`).
- **Boot Command** – Automate the VM boot process using a set of commands (via a VNC connection).
- **Integrated SSH/WinRM Access** – Allows VM provisioning and customization via SSH or WinRM.

## Components

- `kubevirt-iso` - This builder starts from a ISO file and builds virtual machine image on a KubeVirt cluster.

### Design

<img src="docs/kubevirt-iso-builder-design.jpg" alt="Design" width="400"/>

## Installation

### Automatic Installation

Packer supports automatic installation of the Packer plugins.

To install this plugin, copy and paste this code into your Packer configuration:

```hcl
packer {
  required_plugins {
    kubevirt = {
      source  = "github.com/flippyboy/kubevirt"
      version = ">= 0.9.0"
    }
  }
}
```

And now run `packer init` command. The plugin will be installed automatically.

### Manual Installation

Download the latest release from the [Releases](https://github.com/flippyboy/packer-plugin-kubevirt/releases) page and then install the plugin:

```shell
$ packer plugins install --path packer-plugin-kubevirt github.com/flippyboy/kubevirt
```

### Building From Source

Clone the repository and build the plugin from the root directory:

```shell
$ go build -ldflags="-X github.com/flippyboy/packer-plugin-kubevirt/version.Version=0.9.1" -o packer-plugin-kubevirt
```

Then install the compiled plugin:

```shell
$ packer plugins install --path packer-plugin-kubevirt github.com/flippyboy/kubevirt
```

## Releases

Releases are built and published automatically when a version tag is pushed:

1. Update `version/VERSION`, `version/version.go`, and `CHANGELOG.md`
2. Commit and push to `main`
3. Create and push a tag: `git tag vX.Y.Z && git push origin vX.Y.Z`

The [release workflow](.github/workflows/release.yml) runs tests, builds all platform archives, generates `SHA256SUMS`, and publishes the GitHub release.

## Usage

Refer to the usage guidance in the [examples](./examples/builder/kubevirt-iso) of this plugin.
