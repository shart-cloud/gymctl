package environment

import (
	"fmt"
	"strings"

	"gymctl/internal/scenario"
)

type vagrantMachine struct {
	LogicalName string
	VMName      string
	Role        string
	CPUs        int
	MemoryMB    int
	IP          string
}

type bootstrapParams struct {
	KubeVersion   string // minor version for apt repo, e.g. "v1.31"
	PodCIDR       string
	CNI           string
	BootstrapMode string
}

func resolveBootstrapParams(spec *scenario.KubernetesSpec) bootstrapParams {
	p := bootstrapParams{
		KubeVersion:   "v1.31",
		PodCIDR:       "10.244.0.0/16",
		CNI:           "flannel",
		BootstrapMode: VagrantBootstrapManaged,
	}
	if spec == nil || spec.Vagrant == nil {
		return p
	}
	v := spec.Vagrant
	if strings.TrimSpace(v.KubernetesVersion) != "" {
		p.KubeVersion = strings.TrimSpace(v.KubernetesVersion)
	}
	if strings.TrimSpace(v.PodCIDR) != "" {
		p.PodCIDR = strings.TrimSpace(v.PodCIDR)
	}
	if strings.TrimSpace(v.CNI) != "" {
		p.CNI = strings.ToLower(strings.TrimSpace(v.CNI))
	}
	p.BootstrapMode = ResolveVagrantBootstrapMode(spec)
	return p
}

func buildVagrantfile(spec *scenario.KubernetesSpec, machines []vagrantMachine, backend VagrantBackend) string {
	box := "bento/ubuntu-22.04"
	if spec != nil && spec.Vagrant != nil && strings.TrimSpace(spec.Vagrant.Box) != "" {
		box = strings.TrimSpace(spec.Vagrant.Box)
	}

	// Hyper-V box substitution: Bento doesn't publish Hyper-V builds.
	if backend == BackendHyperV && box == "bento/ubuntu-22.04" {
		box = "generic/ubuntu2204"
	}

	bp := resolveBootstrapParams(spec)

	var b strings.Builder
	b.WriteString("Vagrant.configure(\"2\") do |config|\n")
	b.WriteString(fmt.Sprintf("  config.vm.box = %q\n", box))
	// Increase boot timeout for slow Windows/macOS machines (default is 300s).
	b.WriteString("  config.vm.boot_timeout = 600\n")
	b.WriteString("\n")

	for _, m := range machines {
		b.WriteString(fmt.Sprintf("  config.vm.define %q do |node|\n", m.VMName))
		b.WriteString(fmt.Sprintf("    node.vm.hostname = %q\n", m.VMName))

		// Hyper-V doesn't support static IP assignment on private networks.
		if m.IP != "" && backend != BackendHyperV {
			b.WriteString(fmt.Sprintf("    node.vm.network \"private_network\", ip: %q\n", m.IP))
		}

		writeProviderBlock(&b, m, backend)

		b.WriteString(fmt.Sprintf("    node.vm.provision \"shell\", inline: %q\n", "set -e; sudo mkdir -p /etc/jerry-gym; echo role="+m.Role+" | sudo tee /etc/jerry-gym/role >/dev/null"))

		if bp.BootstrapMode == VagrantBootstrapManaged {
			writePreinstallProvisioner(&b, bp)
			if m.Role == "control-plane" {
				writeControlPlaneProvisioner(&b, m, bp, backend)
			} else if m.Role == "worker" {
				writeWorkerJoinProvisioner(&b)
			}
		}

		b.WriteString("  end\n\n")
	}

	b.WriteString("end\n")
	return b.String()
}

func writeProviderBlock(b *strings.Builder, m vagrantMachine, backend VagrantBackend) {
	switch backend {
	case BackendHyperV:
		writeHyperVBlock(b, m)
	case BackendVMware:
		writeVMwareBlock(b, m)
	case BackendLibvirt:
		writeLibvirtBlock(b, m)
	default:
		writeVirtualBoxBlock(b, m)
	}
}

func writeVirtualBoxBlock(b *strings.Builder, m vagrantMachine) {
	b.WriteString("    node.vm.provider \"virtualbox\" do |vb|\n")
	// Set a clean display name in the VirtualBox GUI.
	b.WriteString(fmt.Sprintf("      vb.name = %q\n", m.VMName))
	// Keep VMs headless — required on Windows/macOS where GUI pops up by default.
	b.WriteString("      vb.gui = false\n")
	if m.CPUs > 0 {
		b.WriteString(fmt.Sprintf("      vb.cpus = %d\n", m.CPUs))
	}
	if m.MemoryMB > 0 {
		b.WriteString(fmt.Sprintf("      vb.memory = %d\n", m.MemoryMB))
	}
	// Fix DNS resolution inside VMs on Windows hosts. Without this,
	// apt/curl/kubeadm often fail to resolve hostnames.
	b.WriteString("      vb.customize [\"modifyvm\", :id, \"--natdnshostresolver1\", \"on\"]\n")
	b.WriteString("      vb.customize [\"modifyvm\", :id, \"--natdnsproxy1\", \"on\"]\n")
	b.WriteString("    end\n")
}

func writeHyperVBlock(b *strings.Builder, m vagrantMachine) {
	b.WriteString("    node.vm.provider \"hyperv\" do |hv|\n")
	b.WriteString(fmt.Sprintf("      hv.vmname = %q\n", m.VMName))
	if m.CPUs > 0 {
		b.WriteString(fmt.Sprintf("      hv.cpus = %d\n", m.CPUs))
	}
	if m.MemoryMB > 0 {
		b.WriteString(fmt.Sprintf("      hv.memory = %d\n", m.MemoryMB))
	}
	b.WriteString("      hv.enable_virtualization_extensions = true\n")
	b.WriteString("    end\n")
}

func writeVMwareBlock(b *strings.Builder, m vagrantMachine) {
	b.WriteString("    node.vm.provider \"vmware_desktop\" do |vmw|\n")
	b.WriteString(fmt.Sprintf("      vmw.vmx[\"displayName\"] = %q\n", m.VMName))
	if m.CPUs > 0 {
		b.WriteString(fmt.Sprintf("      vmw.vmx[\"numvcpus\"] = \"%d\"\n", m.CPUs))
	}
	if m.MemoryMB > 0 {
		b.WriteString(fmt.Sprintf("      vmw.vmx[\"memsize\"] = \"%d\"\n", m.MemoryMB))
	}
	b.WriteString("      vmw.gui = false\n")
	b.WriteString("    end\n")
}

func writeLibvirtBlock(b *strings.Builder, m vagrantMachine) {
	b.WriteString("    node.vm.provider \"libvirt\" do |lv|\n")
	if m.CPUs > 0 {
		b.WriteString(fmt.Sprintf("      lv.cpus = %d\n", m.CPUs))
	}
	if m.MemoryMB > 0 {
		b.WriteString(fmt.Sprintf("      lv.memory = %d\n", m.MemoryMB))
	}
	b.WriteString("      lv.driver = \"kvm\"\n")
	b.WriteString("    end\n")
}

// writeInlineProvisioner writes a named shell provisioner using Ruby heredoc syntax.
func writeInlineProvisioner(b *strings.Builder, name string, script string) {
	b.WriteString(fmt.Sprintf("    node.vm.provision \"shell\", name: %q, inline: <<-'SHELL'\n", name))
	b.WriteString(script)
	if !strings.HasSuffix(script, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("    SHELL\n")
}

func writePreinstallProvisioner(b *strings.Builder, bp bootstrapParams) {
	script := fmt.Sprintf(`set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

# Disable swap
swapoff -a
sed -i '/\sswap\s/d' /etc/fstab

# Load required kernel modules
cat > /etc/modules-load.d/k8s.conf <<EOF
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

# Configure sysctl for Kubernetes networking
cat > /etc/sysctl.d/k8s.conf <<EOF
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Install prerequisites (conntrack, socat required by kubeadm preflight)
apt-get update
apt-get install -y ca-certificates curl gnupg conntrack socat
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update
apt-get install -y containerd.io

# Configure containerd with SystemdCgroup
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
systemctl restart containerd
systemctl enable containerd

# Install kubeadm, kubelet, kubectl
KUBE_VERSION=%q
curl -fsSL "https://pkgs.k8s.io/core:/stable:/${KUBE_VERSION}/deb/Release.key" | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/${KUBE_VERSION}/deb/ /" \
  > /etc/apt/sources.list.d/kubernetes.list
apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl
systemctl enable kubelet
`, bp.KubeVersion)

	writeInlineProvisioner(b, "k8s-preinstall", script)
}

func writeControlPlaneProvisioner(b *strings.Builder, m vagrantMachine, bp bootstrapParams, backend VagrantBackend) {
	// Determine the advertise address: use static IP if available, otherwise
	// discover at runtime (Hyper-V doesn't support static IPs).
	advertiseSnippet := ""
	if m.IP != "" && backend != BackendHyperV {
		advertiseSnippet = fmt.Sprintf("ADVERTISE_IP=%q", m.IP)
	} else {
		advertiseSnippet = `ADVERTISE_IP=$(hostname -I | awk '{for(i=1;i<=NF;i++){if($i !~ /^127\./ && $i !~ /^169\.254\./){print $i; exit}}}')`
	}

	cniSnippet := ""
	switch bp.CNI {
	case "calico":
		cniSnippet = `# Install Calico CNI
kubectl --kubeconfig=/etc/kubernetes/admin.conf apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.27.0/manifests/calico.yaml`
	default: // flannel
		cniSnippet = `# Install Flannel CNI
kubectl --kubeconfig=/etc/kubernetes/admin.conf apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml`
	}

	script := fmt.Sprintf(`set -euo pipefail

%s

kubeadm init \
  --apiserver-advertise-address="${ADVERTISE_IP}" \
  --pod-network-cidr=%q

# Save join command for workers (synced folder)
kubeadm token create --print-join-command > /vagrant/join-command.sh
chmod 0644 /vagrant/join-command.sh

# Set up kubectl for vagrant user
mkdir -p /home/vagrant/.kube
cp /etc/kubernetes/admin.conf /home/vagrant/.kube/config
chown -R vagrant:vagrant /home/vagrant/.kube

%s

# Wait for this node to become Ready
echo "Waiting for control-plane node to become Ready..."
for i in $(seq 1 120); do
  if kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes | grep -q ' Ready'; then
    echo "Control-plane node is Ready."
    break
  fi
  sleep 5
done
`, advertiseSnippet, bp.PodCIDR, cniSnippet)

	writeInlineProvisioner(b, "k8s-control-plane-init", script)
}

func writeWorkerJoinProvisioner(b *strings.Builder) {
	script := `set -euo pipefail

# Wait for join command from control-plane (safety net; Vagrant is sequential
# so /vagrant/join-command.sh should already exist).
echo "Waiting for join command..."
for i in $(seq 1 60); do
  if [ -f /vagrant/join-command.sh ]; then
    break
  fi
  sleep 5
done

if [ ! -f /vagrant/join-command.sh ]; then
  echo "ERROR: /vagrant/join-command.sh not found after 5 minutes" >&2
  exit 1
fi

bash /vagrant/join-command.sh
`
	writeInlineProvisioner(b, "k8s-worker-join", script)
}
