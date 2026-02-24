package environment

import (
	"strings"
	"testing"

	"gymctl/internal/scenario"
)

func TestBuildVagrantfile_ManagedMode(t *testing.T) {
	trueVal := true
	spec := &scenario.KubernetesSpec{
		CreateCluster: &trueVal,
		Provider:      "vagrant",
		Vagrant: &scenario.VagrantProviderSpec{
			BootstrapMode: "managed",
			Box:           "bento/ubuntu-22.04",
			Topology:      scenario.VagrantTopologySpec{ControlPlanes: 1, Workers: 1},
			Network:       scenario.VagrantNetworkSpec{PrivateCIDR: "192.168.56.0/24"},
		},
	}
	machines := buildVagrantMachines("test-exercise", spec)
	vf := buildVagrantfile(spec, machines, BackendVirtualBox)

	mustContain := []string{
		"k8s-preinstall",
		"k8s-control-plane-init",
		"k8s-worker-join",
		"kubeadm init",
		"join-command.sh",
		"containerd",
		"flannel",
		"swapoff",
		"br_netfilter",
		"SystemdCgroup",
		"ADVERTISE_IP=\"192.168.56.10\"",
		"10.244.0.0/16",
	}
	for _, want := range mustContain {
		if !strings.Contains(vf, want) {
			t.Errorf("managed Vagrantfile missing %q", want)
		}
	}

	// Control-plane should get preinstall + cp-init, NOT worker-join
	cpSection := extractNodeSection(vf, "test-exercise-cp-1")
	if cpSection == "" {
		t.Fatal("could not find cp-1 section")
	}
	if !strings.Contains(cpSection, "k8s-control-plane-init") {
		t.Error("cp-1 section missing control-plane-init provisioner")
	}
	if strings.Contains(cpSection, "k8s-worker-join") {
		t.Error("cp-1 section should not have worker-join provisioner")
	}

	// Worker should get preinstall + worker-join, NOT cp-init
	workerSection := extractNodeSection(vf, "test-exercise-worker-1")
	if workerSection == "" {
		t.Fatal("could not find worker-1 section")
	}
	if !strings.Contains(workerSection, "k8s-worker-join") {
		t.Error("worker-1 section missing worker-join provisioner")
	}
	if strings.Contains(workerSection, "k8s-control-plane-init") {
		t.Error("worker-1 section should not have control-plane-init provisioner")
	}
}

func TestBuildVagrantfile_BarenodeMode(t *testing.T) {
	trueVal := true
	spec := &scenario.KubernetesSpec{
		CreateCluster: &trueVal,
		Provider:      "vagrant",
		Vagrant: &scenario.VagrantProviderSpec{
			BootstrapMode: "barenode",
			Box:           "bento/ubuntu-22.04",
			Topology:      scenario.VagrantTopologySpec{ControlPlanes: 1, Workers: 1},
			Network:       scenario.VagrantNetworkSpec{PrivateCIDR: "192.168.56.0/24"},
		},
	}
	machines := buildVagrantMachines("test-bare", spec)
	vf := buildVagrantfile(spec, machines, BackendVirtualBox)

	mustNotContain := []string{
		"k8s-preinstall",
		"k8s-control-plane-init",
		"k8s-worker-join",
		"kubeadm init",
		"containerd",
	}
	for _, bad := range mustNotContain {
		if strings.Contains(vf, bad) {
			t.Errorf("barenode Vagrantfile should not contain %q", bad)
		}
	}

	// Role marker should still be present
	if !strings.Contains(vf, "jerry-gym/role") {
		t.Error("barenode Vagrantfile missing role marker provisioner")
	}
}

func TestBuildVagrantfile_HyperVAdvertiseIP(t *testing.T) {
	trueVal := true
	spec := &scenario.KubernetesSpec{
		CreateCluster: &trueVal,
		Provider:      "vagrant",
		Vagrant: &scenario.VagrantProviderSpec{
			BootstrapMode: "managed",
			Topology:      scenario.VagrantTopologySpec{ControlPlanes: 1, Workers: 0},
			Network:       scenario.VagrantNetworkSpec{PrivateCIDR: "192.168.56.0/24"},
		},
	}
	machines := buildVagrantMachines("hyper-test", spec)
	vf := buildVagrantfile(spec, machines, BackendHyperV)

	// Hyper-V should use runtime IP discovery, not static
	if strings.Contains(vf, "ADVERTISE_IP=\"192.168.56.") {
		t.Error("Hyper-V Vagrantfile should not use static ADVERTISE_IP")
	}
	if !strings.Contains(vf, "hostname -I") {
		t.Error("Hyper-V Vagrantfile should discover IP via hostname -I")
	}
}

func TestBuildVagrantfile_CustomKubeVersion(t *testing.T) {
	trueVal := true
	spec := &scenario.KubernetesSpec{
		CreateCluster: &trueVal,
		Provider:      "vagrant",
		Vagrant: &scenario.VagrantProviderSpec{
			BootstrapMode:     "managed",
			KubernetesVersion: "v1.29",
			Topology:          scenario.VagrantTopologySpec{ControlPlanes: 1, Workers: 0},
		},
	}
	machines := buildVagrantMachines("ver-test", spec)
	vf := buildVagrantfile(spec, machines, BackendVirtualBox)

	if !strings.Contains(vf, "v1.29") {
		t.Error("Vagrantfile should use custom Kubernetes version v1.29")
	}
}

func TestBuildVagrantfile_CalicoCNI(t *testing.T) {
	trueVal := true
	spec := &scenario.KubernetesSpec{
		CreateCluster: &trueVal,
		Provider:      "vagrant",
		Vagrant: &scenario.VagrantProviderSpec{
			BootstrapMode: "managed",
			CNI:           "calico",
			Topology:      scenario.VagrantTopologySpec{ControlPlanes: 1, Workers: 0},
		},
	}
	machines := buildVagrantMachines("cni-test", spec)
	vf := buildVagrantfile(spec, machines, BackendVirtualBox)

	if !strings.Contains(vf, "calico") {
		t.Error("Vagrantfile should reference calico CNI")
	}
	if strings.Contains(vf, "flannel") {
		t.Error("Vagrantfile should not reference flannel when calico is selected")
	}
}

func TestResolveBootstrapParams_Defaults(t *testing.T) {
	p := resolveBootstrapParams(nil)
	if p.KubeVersion != "v1.31" {
		t.Errorf("expected default KubeVersion v1.31, got %s", p.KubeVersion)
	}
	if p.PodCIDR != "10.244.0.0/16" {
		t.Errorf("expected default PodCIDR 10.244.0.0/16, got %s", p.PodCIDR)
	}
	if p.CNI != "flannel" {
		t.Errorf("expected default CNI flannel, got %s", p.CNI)
	}
	if p.BootstrapMode != VagrantBootstrapManaged {
		t.Errorf("expected default BootstrapMode managed, got %s", p.BootstrapMode)
	}
}

// extractNodeSection extracts the Vagrant node definition block for a given VM name.
func extractNodeSection(vf, vmName string) string {
	marker := `config.vm.define "` + vmName + `"`
	start := strings.Index(vf, marker)
	if start < 0 {
		return ""
	}
	// Find the matching "end" by counting nesting
	depth := 0
	for i := start; i < len(vf); i++ {
		line := ""
		j := i
		for j < len(vf) && vf[j] != '\n' {
			j++
		}
		line = strings.TrimSpace(vf[i:j])
		if strings.HasSuffix(line, "do |node|") || strings.HasSuffix(line, "do |vb|") || strings.HasSuffix(line, "do |hv|") {
			depth++
		}
		if line == "end" {
			depth--
			if depth <= 0 {
				return vf[start : j+1]
			}
		}
		i = j
	}
	return vf[start:]
}
