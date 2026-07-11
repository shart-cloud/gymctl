# gymctl image pipeline (Packer)

Builds an **Ubuntu 22.04 (Jammy)** image with **Kubernetes pre-baked**, and emits
two artifacts from the same baked disk:

1. a **libvirt Vagrant box** — `output/gymctl-ubuntu2204-k8s-<ver>.libvirt.box`
2. a **KubeVirt containerDisk** OCI image — `gymctl/ubuntu2204-k8s-<ver>-containerdisk:latest`
   (built locally, **emit-only — not pushed, no consumer yet**)

The build uses the Packer **`qemu`** builder with KVM acceleration, headless.
It needs only `qemu` + `/dev/kvm` + `xorriso` + `docker` — **no libvirt, no virsh,
no sudo, no cloud-localds/genisoimage**.

## Prerequisites

- `packer` (>= 1.15), `qemu-img`, `qemu-system-x86_64`, `xorriso`, `docker`
- `/dev/kvm` present and your user in the `kvm` group (KVM acceleration)

## Build

```bash
cd packer
./build.sh                       # bakes the default k8s channel (v1.36)
./build.sh -var k8s_version=v1.34   # bake a different channel
```

`build.sh` downloads + checksum-verifies the Ubuntu cloud image, runs
`packer init` (installs the qemu plugin), then `packer build -force`.

Prefer running Packer directly? The base image must already exist first
(build.sh fetches it into `output/`):

```bash
packer init .
packer validate .
packer build -force .
```

### The `k8s_version` variable

`k8s_version` is the **pkgs.k8s.io stable channel** (e.g. `v1.36`), not a full
patch. The channel resolves to the newest patch in the series at build time.

- Default: **`v1.36`** — the newest published stable channel. `v1.37` is not yet
  published on pkgs.k8s.io (returns 403).
- The bake script prints the exact `kubeadm`/`kubelet`/`kubectl` version installed
  (currently `1.36.2`) and the pre-pulled control-plane images.

## What gets baked in

- **containerd** (`containerd.io` from Docker's apt repo), `config.toml` generated
  with `SystemdCgroup = true`, service enabled.
- Kernel prereqs: `overlay` + `br_netfilter` modules, the k8s sysctls
  (`bridge-nf-call-iptables=1`, `ip_forward=1`), **swap off** (+ fstab).
- **kubelet / kubeadm / kubectl** from `pkgs.k8s.io`, `apt-mark hold`-ed.
- **Pre-pulled control-plane images** (`kubeadm config images pull`) baked into
  containerd's `k8s.io` namespace — the main first-boot speedup.
- User **`vagrant`** with passwordless sudo and the well-known **Vagrant insecure
  SSH public key** in `~/.ssh/authorized_keys`. SSH enabled.

`kubeadm init` is **not** run — bootstrap stays at boot time for topology
flexibility (single control-plane, multi-node, etc.).

## Layout

```
packer/
├── build.sh                    # blessed entrypoint (download → init → build)
├── gymctl-k8s.pkr.hcl          # qemu builder + provisioners + post-processors
├── cloud-init/
│   ├── user-data               # NoCloud: creates vagrant user + insecure key
│   └── meta-data
├── vagrant_insecure_key(.pub)  # canonical Vagrant insecure keypair (public)
├── scripts/
│   ├── 10-k8s.sh               # the Kubernetes bake
│   ├── 90-cleanup.sh           # de-provision + zero-fill so the box is small
│   ├── package-box.sh          # tar the qcow2 into a vagrant-libvirt .box
│   └── build-containerdisk.sh  # docker build the KubeVirt containerDisk
├── containerdisk/Dockerfile    # FROM scratch; ADD disk.qcow2 /disk/
└── output/                     # gitignored — base image + artifacts land here
```

The cloud-init NoCloud seed is produced from `cloud-init/{user-data,meta-data}`
via Packer's `cd_files` (which shells out to **`xorriso`** on Linux) and attached
as a CD-ROM labeled `cidata`.

### How the libvirt box is packaged

Packer's built-in `vagrant` post-processor has **no libvirt provider**, so the box
is assembled by hand (`scripts/package-box.sh`): a gzip'd tar of
`box.img` (the qcow2), `metadata.json`
(`{"provider":"libvirt","format":"qcow2","virtual_size":<GB>}`) and a minimal
`Vagrantfile`.

## Using the libvirt box (follow-on)

Requires the `vagrant-libvirt` plugin (installed later, not needed to build):

```bash
vagrant plugin install vagrant-libvirt
vagrant box add gymctl/ubuntu2204-k8s output/gymctl-ubuntu2204-k8s-1.36.libvirt.box
```

Then in a `Vagrantfile`: `config.vm.box = "gymctl/ubuntu2204-k8s"`, provider
`:libvirt`. `vagrant up` SSHes in as **`vagrant`** using the insecure key baked in.
Bootstrap the cluster on the running VM with `kubeadm init` / `kubeadm join`.

Facts the consumer needs:
- box name to add: `gymctl/ubuntu2204-k8s`
- internal disk path inside the box: `box.img` (qcow2)
- login: user `vagrant`, well-known Vagrant insecure key (`vagrant`/`vagrant`
  password fallback)
- k8s is baked but **not** initialized — run `kubeadm init` at boot.

## KubeVirt containerDisk (emit-only)

`scripts/build-containerdisk.sh` builds `gymctl/ubuntu2204-k8s-<ver>-containerdisk:latest`
from the same qcow2 (`FROM scratch`, disk at `/disk/disk.qcow2`). It is **built
locally and not pushed** — there is no gymctl consumer for it yet.
