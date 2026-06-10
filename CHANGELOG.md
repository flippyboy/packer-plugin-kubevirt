## 0.9.2

### BUG FIXES:

* fix: keep SSH/WinRM port-forward listener alive after transient KubeVirt API errors (e.g. guest not ready yet)
* fix: close port-forward listener on step cleanup

## 0.9.1

### IMPROVEMENTS:

* feat: add `sysprep_content` and `sysprep_files` for KubeVirt sysprep volume (templated inline content)
* ci: publish GitHub releases via tag-triggered workflow

## 0.9.0

Fork maintained at [flippyboy/packer-plugin-kubevirt](https://github.com/flippyboy/packer-plugin-kubevirt).

### IMPROVEMENTS:

* feat: add `storage_class` option for plugin-created DataVolumes
* feat: add `cd_content` option for inline install media (superseded by `sysprep_content` in 0.9.1)

## 0.8.0
Migrated codebase from [kv-infra/packer-plugin-kubevirt](https://github.com/kv-infra/packer-plugin-kubevirt)
### IMPROVEMENTS:

* feat: create artifact from builder
  Create an artifact from the builder that
  could be used to trigger the post-processors.
  [GH-4](https://github.com/hashicorp/packer-plugin-kubevirt/pull/4)


### BUG FIXES:

* fix: typo in log messages of VM creation
  Changed from 'VirutalMachine' to 'VirtualMachine'.
  [GH-4](https://github.com/hashicorp/packer-plugin-kubevirt/pull/4)
  
* fix: avoid crash if kubeconfig is not set
  Packer crashes if the KubeConfig environment
  variable is not, instead it should just show an
  error and ask the user to set this variable.
  [GH-4](https://github.com/hashicorp/packer-plugin-kubevirt/pull/4)

