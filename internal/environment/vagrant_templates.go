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

func buildVagrantfile(spec *scenario.KubernetesSpec, machines []vagrantMachine) string {
	box := "bento/ubuntu-22.04"
	if spec != nil && spec.Vagrant != nil && strings.TrimSpace(spec.Vagrant.Box) != "" {
		box = strings.TrimSpace(spec.Vagrant.Box)
	}

	var b strings.Builder
	b.WriteString("Vagrant.configure(\"2\") do |config|\n")
	b.WriteString(fmt.Sprintf("  config.vm.box = %q\n", box))
	// Increase boot timeout for slow Windows/macOS machines (default is 300s).
	b.WriteString("  config.vm.boot_timeout = 600\n")
	b.WriteString("\n")

	for _, m := range machines {
		b.WriteString(fmt.Sprintf("  config.vm.define %q do |node|\n", m.VMName))
		b.WriteString(fmt.Sprintf("    node.vm.hostname = %q\n", m.VMName))
		if m.IP != "" {
			b.WriteString(fmt.Sprintf("    node.vm.network \"private_network\", ip: %q\n", m.IP))
		}
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
		b.WriteString(fmt.Sprintf("    node.vm.provision \"shell\", inline: %q\n", "set -e; sudo mkdir -p /etc/jerry-gym; echo role="+m.Role+" | sudo tee /etc/jerry-gym/role >/dev/null"))
		b.WriteString("  end\n\n")
	}

	b.WriteString("end\n")
	return b.String()
}
