# AGENTS.md

Guidance for AI coding agents working in **packer-plugin-kubevirt**.

## Project summary

This repository is a [HashiCorp Packer](https://developer.hashicorp.com/packer) plugin that builds KubeVirt VM images inside a Kubernetes cluster from an existing ISO DataVolume. It is a fork of [hashicorp/packer-plugin-kubevirt](https://github.com/hashicorp/packer-plugin-kubevirt), maintained at [github.com/flippyboy/packer-plugin-kubevirt](https://github.com/flippyboy/packer-plugin-kubevirt).

- **Module:** `github.com/flippyboy/packer-plugin-kubevirt`
- **Packer plugin source:** `github.com/flippyboy/kubevirt`
- **Current version:** see `version/VERSION` (latest release: v0.9.2)
- **Status:** under development; not production-ready
- **License:** MPL-2.0

### What the plugin does

The `kubevirt-iso` builder:

1. Validates a pre-existing ISO DataVolume in the target namespace
2. Creates a temporary VM, installs the OS from ISO
3. Optionally provisions the guest over SSH or WinRM
4. Stops the VM and clones the root disk into a bootable DataVolume artifact

Supported guest OS types: **linux** (default) and **windows**.

---

## Packer plugin architecture

Packer plugins follow the [Packer Plugin SDK](https://github.com/hashicorp/packer-plugin-sdk) model. Understanding this split is essential when changing behavior.

### Packer Core (RPC client)

- Parses HCL/JSON templates and interpolates variables
- Discovers plugins via `packer init` and the plugin registry manifest
- Spawns plugin processes and passes validated configuration over RPC
- Orchestrates the full build (builders → provisioners → post-processors)

### Plugin process (RPC server)

- Runs as a separate binary (`packer-plugin-kubevirt`)
- Registers components with `plugin.NewSet()` in `main.go`
- Implements domain logic: VM lifecycle, KubeVirt API calls, artifacts
- Returns validation results, artifacts, and success/failure to Core

### Standard builder interface

Every builder must implement three methods (see [packer-plugin-scaffolding](https://github.com/hashicorp/packer-plugin-scaffolding)):

| Method | Purpose |
|--------|---------|
| `ConfigSpec() hcldec.ObjectSpec` | HCL2 decoding schema for template blocks |
| `Prepare(...)` | Decode config, validate, set up clients; may return generated variables |
| `Run(ctx, ui, hook)` | Execute the build; return a `packer.Artifact` or error |

Builders typically use the **multistep** pattern: a slice of `multistep.Step` values run in order via `commonsteps.NewRunner`, with shared state in a `multistep.BasicStateBag`.

### Plugin registration (this repo)

```go
// main.go
setup := plugin.NewSet()
setup.RegisterBuilder("iso", new(iso.Builder))
setup.SetVersion(version.PluginVersion)
setup.Run()
```

- Registered name `iso` + plugin name `kubevirt` → HCL type **`kubevirt-iso`**
- Artifact builder ID: `kubevirt.iso` (see `builder/kubevirt/iso/artifact.go`)

### Config and code generation

Builder config lives in `builder/kubevirt/iso/config.go` and embeds `common.PackerConfig` for standard Packer fields (`communicator`, provisioner settings, etc.).

```go
//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config,Network,...
```

- `packer-sdc mapstructure-to-hcl2` generates `config.hcl2spec.go`
- `packer-sdc struct-markdown` generates docs partials under `docs-partials/`
- Run `make generate` after changing config struct fields or doc comments

### Release artifact naming

Packer expects GitHub release assets named:

```
packer-plugin-kubevirt_v{VERSION}_x5.0_{os}_{arch}.zip
packer-plugin-kubevirt_v{VERSION}_SHA256SUMS
packer-plugin-kubevirt_v{VERSION}_manifest.json
```

The `x5.0` segment is the plugin API protocol version from `go run . describe`.

---

## Repository layout

```
main.go                          # Plugin entrypoint; registers builders
manifest.json                    # Plugin protocol metadata template
version/                         # VERSION file + version.go
builder/kubevirt/
  common/portforwarder.go        # Local TCP listener → KubeVirt API tunnel
  iso/
    builder.go                   # Multistep orchestration
    config.go                    # HCL config + validation
    config.hcl2spec.go           # Generated HCL2 spec (do not hand-edit)
    artifact.go                  # Build output (bootable DataVolume name)
    resources.go                 # VM/ConfigMap/PVC object definitions
    step_*.go                    # Individual multistep steps
docs/                            # Authoritative builder docs (MDX)
docs-partials/                   # Generated config reference partials
examples/builder/kubevirt-iso/   # Working Packer templates (fedora, rhel, windows)
.github/workflows/release.yml    # Tag-triggered release pipeline
GNUmakefile                      # build, test, dev, generate, plugin-check
```

There are **no provisioners or post-processors** in this plugin today—only the `kubevirt-iso` builder.

---

## Build pipeline (multistep flow)

Defined in `builder/kubevirt/iso/builder.go`:

| Step | File | Role |
|------|------|------|
| Validate ISO DataVolume | `step_validate_iso_datavolume.go` | Ensure `iso_volume_name` exists |
| Copy media files | `step_copy_media_files.go` | Create ConfigMap from `media_files` / sysprep content |
| Create VM | `step_create_virtualmachine.go` | Create VM + root PVC; wait for `VM.Status.Ready` |
| Boot command | `step_boot_command.go` | VNC keystrokes via KubeVirt VMI subresource |
| Wait for installation | `step_wait_for_installation.go` | **Fixed sleep** (`installation_wait_timeout`) |
| Start port-forward | `step_start_portforward.go` | Bind local port for SSH/WinRM (if communicator set) |
| Connect communicator | Packer SDK `communicator.StepConnect` | Retry until SSH/WinRM works |
| Provision | Packer SDK `commonsteps.StepProvision` | Run template provisioners |
| Stop VM | `step_stop_virtualmachine.go` | Stop temporary VM |
| Create bootable volume | `step_create_bootablevolume.go` | CDI clone of root disk → artifact |

### OS-specific media handling (`resources.go`)

| `os_type` | Install media | Volume type |
|-----------|---------------|-------------|
| `linux` | `media_files` | ConfigMap → OEMDRV CD (`VolumeLabel: OEMDRV`) |
| `windows` | `sysprep_files`, `sysprep_content`, `media_files` | KubeVirt sysprep volume (`autounattend.xml`, scripts) |

Windows-only options (`sysprep_files`, `sysprep_content`) are rejected in `Config.Prepare` when `os_type != "windows"`.

---

## Guest access: SSH / WinRM via Kubernetes API

The plugin does **not** connect to the guest VM IP directly. Communicator traffic is tunneled through the Kubernetes API.

### Data path

```
Packer communicator
  → localhost:ssh_local_port / winrm_local_port  (default bind: 127.0.0.1)
  → builder/kubevirt/common/portforwarder.go (local TCP listener)
  → KubeVirt subresource WebSocket:
      /apis/subresources.kubevirt.io/v1/namespaces/{ns}/virtualmachines/{name}/portforward/{port}/tcp
  → virt-handler dials guest pod-network IP:remote_port
  → SSH (22) or WinRM (5985) inside the guest
```

Equivalent to `virtctl port-forward vm <name> <local>:<remote>`.

### Required HCL settings

```hcl
communicator = "winrm"   # or "ssh"
winrm_host       = "127.0.0.1"
winrm_local_port = 5000
winrm_remote_port = 5985
winrm_username   = "Administrator"
winrm_password   = "..."
winrm_wait_timeout = "25m"
installation_wait_timeout = "20m"  # blind wait before communicator starts
```

`ssh_*` mirrors the same pattern (`ssh_host`, `ssh_local_port`, `ssh_remote_port`).

### WinRM availability detection

The plugin does **not** probe guest readiness from Kubernetes. After `installation_wait_timeout`:

1. `StepStartPortForward` binds the local listener
2. Packer SDK `StepConnectWinRM` retries every ~5s until:
   - TCP to localhost succeeds through the tunnel
   - WinRM auth succeeds
   - A PowerShell echo command returns `"WinRM connected."`
3. Overall timeout: `winrm_wait_timeout` (SDK default 30m)

**Important:** `installation_wait_timeout` is not a WinRM health check. If it expires before Windows finishes setup, early port-forward attempts may see `no route to host` from KubeVirt (guest network not ready). Since v0.9.2, transient API errors no longer kill the local listener.

### Windows guest requirements

See `examples/builder/kubevirt-iso/windows/`:

- `autounattend.xml` — unattended install + `auditUser` scripts
- `enable-winrm.ps1` — enable WinRM HTTP on 5985, firewall rule
- `set-network.ps1` — set network profile to Private (WinRM policy)
- Scripts are invoked from the sysprep CD (example uses `F:\` — verify drive letter matches your image)

### RBAC

The build service account needs permission to use KubeVirt port-forward subresources, e.g.:

```yaml
- apiGroups: ["subresources.kubevirt.io"]
  resources: ["virtualmachines/portforward", "virtualmachineinstances/portforward"]
  verbs: ["update"]
```

Also needs standard KubeVirt VM/VMI, CDI DataVolume, and ConfigMap permissions.

---

## Development commands

```bash
# Build plugin binary
make build
# or dev install into Packer plugin directory
make dev

# Run unit tests (Ginkgo in iso/, stdlib in common/)
make test

# Regenerate HCL2 specs and docs partials (requires packer-sdc)
make generate

# Validate plugin binary against SDK expectations
make plugin-check

# Debug a template build
PACKER_LOG=1 packer build examples/builder/kubevirt-iso/fedora/fedora.pkr.hcl
```

### Local iteration

1. `make dev` — builds with `VersionPrerelease=dev` and installs via `packer plugins install --path`
2. Set `KUBECONFIG` to a cluster with KubeVirt + CDI
3. Pre-create the ISO DataVolume referenced by `iso_volume_name`

### Testing conventions

- **Framework:** Ginkgo/Gomega in `builder/kubevirt/iso/*_test.go`; stdlib tests in `common/`
- **KubeVirt client:** tests mock `kubecli` via `kubevirtfake` and gomock
- **Port forward:** `step_start_portforward_test.go` injects `ForwarderFunc` to avoid real API calls
- Run `go test ./...` before committing; CI runs the same on every PR and release

### Changing config fields

1. Edit struct + doc comments in `config.go`
2. Run `make generate`
3. Update examples and hand-written docs if behavior changed
4. Add/adjust validation in `Config.Prepare`

---

## Releases

Tag-triggered workflow (`.github/workflows/release.yml`):

1. Bump `version/VERSION`, `version/version.go`, and `CHANGELOG.md`
2. Commit and push to `main`
3. `git tag vX.Y.Z && git push origin vX.Y.Z`

The workflow validates version files match the tag, runs tests, builds the platform matrix, publishes `SHA256SUMS` + manifest, and creates the GitHub release.

Users install via:

```hcl
packer {
  required_plugins {
    kubevirt = {
      source  = "github.com/flippyboy/kubevirt"
      version = ">= 0.9.2"
    }
  }
}
```

---

## Common pitfalls (from production debugging)

| Symptom | Likely cause |
|---------|----------------|
| `Waiting for WinRM to become available...` forever | Guest WinRM not enabled; wrong credentials; `installation_wait_timeout` too short; sysprep scripts failed (wrong drive letter) |
| `dial tcp ...:5985: no route to host` | Guest network not up yet (expected during install); should be transient after v0.9.2 |
| `connection reset by peer` on `localhost:5000` | Pre-v0.9.2 bug: first port-forward API error killed the listener |
| `packer init` fails | Release zip or SHA256SUMS naming mismatch; wrong `source` hostname |
| Sysprep scripts never run | Cached answer file from prior sysprep on base image ([KubeVirt docs](https://kubevirt.io/user-guide/user_workloads/startup_scripts/)) |

---

## Key dependencies

| Package | Use |
|---------|-----|
| `github.com/hashicorp/packer-plugin-sdk` | Plugin framework, multistep, communicator, config decode |
| `kubevirt.io/client-go` | KubeVirt API (VM, VMI, port-forward, VNC subresources) |
| `kubevirt.io/api` | KubeVirt CRD types |
| `k8s.io/client-go` | Kubernetes clientset (ConfigMaps, etc.) |
| `kubevirt.io/containerized-data-importer-api` | CDI DataVolume types |
| `github.com/mitchellh/go-vnc` | Boot command automation over VNC WebSocket |

Go version: see `.go-version` / `go.mod` (currently Go 1.24.x).

---

## Agent guidelines

- **Stay focused:** only change code required by the task; match existing patterns in `builder/kubevirt/iso/`.
- **Do not hand-edit** `config.hcl2spec.go`; regenerate with `make generate`.
- **Preserve copyright headers** (Red Hat / IBM / MPL-2.0) in Go files.
- **Test changes:** run `go test ./...` after modifying builder logic.
- **Windows vs Linux:** check `os_type` code paths in `resources.go` and `Config.Prepare`.
- **Communicator changes:** consider both the port-forward layer (`common/portforwarder.go`) and Packer SDK `communicator.StepConnect` behavior.
- **Docs:** builder reference lives in `docs/` + `docs-partials/`; examples in `examples/builder/kubevirt-iso/`.
- **Fork URLs:** use `github.com/flippyboy/...` in module paths, plugin source, and release workflows—not `hashicorp/`.

## Reference links

- [Packer plugin overview](https://developer.hashicorp.com/packer/docs/plugins)
- [Packer plugin creation guide](https://developer.hashicorp.com/packer/docs/plugins/creation)
- [Packer Plugin SDK](https://github.com/hashicorp/packer-plugin-sdk)
- [Packer plugin scaffolding template](https://github.com/hashicorp/packer-plugin-scaffolding)
- [KubeVirt startup scripts / sysprep](https://kubevirt.io/user-guide/user_workloads/startup_scripts/)
- [Plugin `README.md`](./README.md) and [examples](./examples/builder/kubevirt-iso/)