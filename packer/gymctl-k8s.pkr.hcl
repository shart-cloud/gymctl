##############################################################################
# gymctl — Kubernetes learning gym image pipeline
#
# Builds an Ubuntu 22.04 (Jammy) cloud image with Kubernetes pre-baked, using
# the Packer *qemu* builder (KVM accel, headless — NO libvirt/virsh required).
#
# Emits two artifacts (see build{} post-processors):
#   1. a libvirt Vagrant box  -> output/gymctl-ubuntu2204-k8s-<ver>.libvirt.box
#   2. a KubeVirt containerDisk OCI image (built locally, not pushed)
#
# The cloud-init NoCloud seed ISO is produced from cloud-init/{user-data,meta-data}
# via Packer's cd_files (which shells out to `xorriso` on this host) and is
# attached as a CD-ROM labeled "cidata" so first boot provisions the vagrant user.
##############################################################################

packer {
  required_plugins {
    qemu = {
      source  = "github.com/hashicorp/qemu"
      version = ">= 1.1.0"
    }
  }
}

variable "k8s_version" {
  type = string
  # pkgs.k8s.io channel. Newest published stable at build time is v1.36
  # (v1.37 is not yet published -> 403). This channel resolves to the newest
  # patch in the series (currently 1.36.2).
  default     = "v1.36"
  description = "Kubernetes pkgs.k8s.io stable channel to bake (e.g. v1.36)."
}

variable "base_image" {
  type        = string
  default     = "./output/jammy-server-cloudimg-amd64.img"
  description = "Path to the downloaded Ubuntu 22.04 cloud image qcow2 (fetched + verified by build.sh)."
}

variable "base_image_checksum" {
  type        = string
  default     = "sha256:d14a3dd35eb6d53a8b088bea2e9fbb0587b5761015c2ce5046c72d158d9cecb3"
  description = "SHA256 of the Ubuntu cloud image (from the published SHA256SUMS)."
}

variable "disk_size" {
  type        = string
  default     = "20480"
  description = "Target disk size in MB (the 2.2G cloud image is grown to this)."
}

variable "cpus" {
  type    = number
  default = 2
}

variable "memory" {
  type    = number
  default = 2048
}

locals {
  # Short version tag used in artifact names, e.g. v1.36 -> 1.36
  ver_tag       = trimprefix(var.k8s_version, "v")
  box_name      = "gymctl-ubuntu2204-k8s-${local.ver_tag}.libvirt.box"
  containerdisk = "gymctl/ubuntu2204-k8s-${local.ver_tag}-containerdisk:latest"
}

source "qemu" "ubuntu2204" {
  # Use the pre-downloaded, checksum-verified cloud image as the base disk.
  iso_url      = var.base_image
  iso_checksum = var.base_image_checksum
  disk_image   = true
  format       = "qcow2"

  # KVM acceleration (host has /dev/kvm and we are in the kvm group).
  accelerator = "kvm"
  headless    = true

  cpus      = var.cpus
  memory    = var.memory
  disk_size = var.disk_size

  # Compact the output disk at the end (qemu-img convert -c). Combined with the
  # zero-fill in the cleanup script this keeps the resulting box small.
  disk_compression    = true
  disk_interface      = "virtio"
  net_device          = "virtio-net"
  use_default_display = false

  # Faster CPU passthrough for the build.
  qemuargs = [
    ["-cpu", "host"]
  ]

  # NoCloud cloud-init seed: user-data + meta-data burned into a "cidata"
  # labeled ISO. On Linux, Packer's cd builder invokes `xorriso`.
  cd_files = [
    "./cloud-init/user-data",
    "./cloud-init/meta-data",
  ]
  cd_label = "cidata"

  # SSH in as the vagrant user (created by cloud-init) with the well-known
  # Vagrant insecure private key.
  ssh_username         = "vagrant"
  ssh_private_key_file = "./vagrant_insecure_key"
  ssh_timeout          = "25m"

  # cloud-init finishes then we shut down cleanly.
  shutdown_command = "sudo shutdown -P now"
  shutdown_timeout = "5m"

  output_directory = "./output/qemu"
  vm_name          = "box.img"
}

build {
  name    = "gymctl-k8s"
  sources = ["source.qemu.ubuntu2204"]

  # Wait for first-boot cloud-init (user creation, growpart) to finish.
  provisioner "shell" {
    execute_command = "chmod +x {{ .Path }}; sudo -E bash '{{ .Path }}'"
    inline = [
      "cloud-init status --wait || true",
      "echo 'cloud-init done'; df -h /",
    ]
  }

  # Bake Kubernetes (containerd + kubeadm/kubelet/kubectl + pre-pulled images).
  provisioner "shell" {
    execute_command  = "chmod +x {{ .Path }}; sudo -E bash '{{ .Path }}'"
    environment_vars = ["K8S_VERSION=${var.k8s_version}"]
    script           = "./scripts/10-k8s.sh"
  }

  # Make the image bootable as a vagrant-libvirt box (no cloud-init at runtime):
  # NIC-agnostic DHCP netplan + SSH host-key regeneration on boot.
  provisioner "shell" {
    execute_command = "chmod +x {{ .Path }}; sudo -E bash '{{ .Path }}'"
    script          = "./scripts/20-vagrant.sh"
  }

  # Cleanup / de-provision so the box is generic and compresses small.
  provisioner "shell" {
    execute_command = "chmod +x {{ .Path }}; sudo -E bash '{{ .Path }}'"
    script          = "./scripts/90-cleanup.sh"
  }

  # Package the qcow2 into a vagrant-libvirt .box (manual tar — Packer's vagrant
  # post-processor has no libvirt provider).
  post-processor "shell-local" {
    environment_vars = [
      "BOX_NAME=${local.box_name}",
      "K8S_VERSION=${var.k8s_version}",
    ]
    scripts = ["./scripts/package-box.sh"]
  }

  # Build the KubeVirt containerDisk OCI image locally (emit-only, not pushed).
  post-processor "shell-local" {
    environment_vars = [
      "CONTAINERDISK_IMAGE=${local.containerdisk}",
    ]
    scripts = ["./scripts/build-containerdisk.sh"]
  }
}
