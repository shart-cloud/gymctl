package environment

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"gymctl/internal/scenario"
)

// VagrantBackend identifies a Vagrant hypervisor provider.
type VagrantBackend string

const (
	BackendVirtualBox VagrantBackend = "virtualbox"
	BackendHyperV     VagrantBackend = "hyperv"
	BackendVMware     VagrantBackend = "vmware_desktop"
	BackendLibvirt    VagrantBackend = "libvirt"
)

// BackendDescriptor holds display metadata for a backend.
type BackendDescriptor struct {
	DisplayName      string
	SupportsStaticIP bool
}

var backendRegistry = map[VagrantBackend]BackendDescriptor{
	BackendVirtualBox: {DisplayName: "VirtualBox", SupportsStaticIP: true},
	BackendHyperV:     {DisplayName: "Hyper-V", SupportsStaticIP: false},
	BackendVMware:     {DisplayName: "VMware Desktop", SupportsStaticIP: true},
	BackendLibvirt:    {DisplayName: "libvirt/KVM", SupportsStaticIP: true},
}

// ResolveVagrantBackend determines which Vagrant backend to use via a
// three-level waterfall: env var > task YAML > OS auto-detection.
func ResolveVagrantBackend(spec *scenario.VagrantProviderSpec) (VagrantBackend, error) {
	// 1. Environment variable override (highest priority).
	if env := strings.TrimSpace(os.Getenv("GYMCTL_VAGRANT_BACKEND")); env != "" {
		b := VagrantBackend(strings.ToLower(env))
		if _, ok := backendRegistry[b]; !ok {
			return "", fmt.Errorf("unsupported vagrant backend %q (set via GYMCTL_VAGRANT_BACKEND)\nSupported: virtualbox, hyperv, vmware_desktop, libvirt", env)
		}
		return b, nil
	}

	// 2. Task YAML spec.
	if spec != nil && strings.TrimSpace(spec.Backend) != "" {
		b := VagrantBackend(strings.ToLower(strings.TrimSpace(spec.Backend)))
		if _, ok := backendRegistry[b]; !ok {
			return "", fmt.Errorf("unsupported vagrant backend %q in task YAML\nSupported: virtualbox, hyperv, vmware_desktop, libvirt", spec.Backend)
		}
		return b, nil
	}

	// 3. Auto-detect by OS/arch.
	return detectBackendForOS()
}

// ValidateBackend checks that the chosen backend's prerequisites are met and
// returns an actionable error with install hints on failure.
func ValidateBackend(backend VagrantBackend) error {
	// Vagrant itself is always required.
	if _, err := exec.LookPath("vagrant"); err != nil {
		var hint string
		switch runtime.GOOS {
		case "windows":
			hint = "Install Vagrant from https://developer.hashicorp.com/vagrant/downloads\n" +
				"After installation, restart your terminal so the PATH is updated."
		case "darwin":
			hint = "Install Vagrant: brew install --cask vagrant\n" +
				"Or download from https://developer.hashicorp.com/vagrant/downloads"
		default:
			hint = "Install Vagrant from https://developer.hashicorp.com/vagrant/downloads"
		}
		return fmt.Errorf("vagrant not found in PATH\n\n%s", hint)
	}

	switch backend {
	case BackendVirtualBox:
		return validateVirtualBox()
	case BackendHyperV:
		return validateHyperV()
	case BackendVMware:
		return validateVMware()
	case BackendLibvirt:
		return validateLibvirt()
	default:
		return fmt.Errorf("unsupported vagrant backend: %s", backend)
	}
}

func validateVirtualBox() error {
	if findVBoxManage() == "" {
		var hint string
		switch runtime.GOOS {
		case "windows":
			hint = "Install VirtualBox from https://www.virtualbox.org/wiki/Downloads\n" +
				"After installation, restart your terminal so the PATH is updated."
		case "darwin":
			hint = "Install VirtualBox: brew install --cask virtualbox\n" +
				"Or download from https://www.virtualbox.org/wiki/Downloads"
		default:
			hint = "Install VirtualBox from https://www.virtualbox.org/wiki/Downloads"
		}
		return fmt.Errorf("VBoxManage not found — is VirtualBox installed?\n\n%s", hint)
	}
	return nil
}

// findVBoxManage returns the path to VBoxManage, checking PATH first then
// well-known install locations. Returns "" if not found.
func findVBoxManage() string {
	name := "VBoxManage"
	if runtime.GOOS == "windows" {
		name = "VBoxManage.exe"
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Check common install locations not always in PATH.
	candidates := vboxSearchPaths()
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// vboxSearchPaths returns well-known VirtualBox install locations per OS.
func vboxSearchPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			`C:\Program Files\Oracle\VirtualBox\VBoxManage.exe`,
			`C:\Program Files (x86)\Oracle\VirtualBox\VBoxManage.exe`,
			`C:\Program Files\VirtualBox\VBoxManage.exe`,
		}
	case "darwin":
		return []string{
			"/usr/local/bin/VBoxManage",
			"/Applications/VirtualBox.app/Contents/MacOS/VBoxManage",
		}
	default:
		return nil
	}
}

func validateHyperV() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Hyper-V backend is only available on Windows")
	}
	// Check that the Hyper-V feature is enabled via PowerShell.
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State").Output()
	if err != nil || !strings.Contains(strings.TrimSpace(string(out)), "Enabled") {
		return fmt.Errorf("Hyper-V is not enabled\n\nEnable it with: Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All\nThen restart your computer")
	}
	if !isWindowsAdmin() {
		return fmt.Errorf("Hyper-V requires an administrator terminal\n\nRight-click your terminal and select \"Run as administrator\"")
	}
	return nil
}

func validateVMware() error {
	found := false
	if _, err := exec.LookPath("vmrun"); err == nil {
		found = true
	}
	if runtime.GOOS == "darwin" && !found {
		// Check common VMware Fusion path on macOS.
		if _, err := os.Stat("/Applications/VMware Fusion.app"); err == nil {
			found = true
		}
	}
	if !found {
		hint := "Install VMware Workstation/Fusion from https://www.vmware.com/"
		if runtime.GOOS == "darwin" {
			hint = "Install VMware Fusion: brew install --cask vmware-fusion\n" +
				"Or download from https://www.vmware.com/products/fusion.html"
		}
		return fmt.Errorf("VMware not found (vmrun not in PATH)\n\n%s", hint)
	}
	if !vagrantPluginInstalled("vagrant-vmware-desktop") {
		return fmt.Errorf("vagrant-vmware-desktop plugin not installed\n\nInstall it with: vagrant plugin install vagrant-vmware-desktop")
	}
	return nil
}

func validateLibvirt() error {
	if _, err := exec.LookPath("virsh"); err != nil {
		return fmt.Errorf("virsh not found in PATH — is libvirt installed?\n\nInstall libvirt:\n  Debian/Ubuntu: sudo apt install libvirt-daemon-system\n  Fedora: sudo dnf install libvirt")
	}
	if !vagrantPluginInstalled("vagrant-libvirt") {
		return fmt.Errorf("vagrant-libvirt plugin not installed\n\nInstall it with: vagrant plugin install vagrant-libvirt")
	}
	return nil
}

// detectBackendForOS probes the current system for available hypervisors.
func detectBackendForOS() (VagrantBackend, error) {
	switch runtime.GOOS {
	case "windows":
		if findVBoxManage() != "" {
			return BackendVirtualBox, nil
		}
		// Fall back to Hyper-V (common on modern Windows where VBox isn't installed).
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"(Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State").Output()
		if err == nil && strings.Contains(strings.TrimSpace(string(out)), "Enabled") {
			return BackendHyperV, nil
		}
		return "", fmt.Errorf("no supported Vagrant backend detected on Windows\n\nInstall VirtualBox from https://www.virtualbox.org/wiki/Downloads\nOr enable Hyper-V: Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All")

	case "darwin":
		if runtime.GOARCH == "arm64" {
			// Apple Silicon — prefer VMware (VirtualBox ARM support is limited).
			if _, err := exec.LookPath("vmrun"); err == nil && vagrantPluginInstalled("vagrant-vmware-desktop") {
				return BackendVMware, nil
			}
			if _, err := os.Stat("/Applications/VMware Fusion.app"); err == nil && vagrantPluginInstalled("vagrant-vmware-desktop") {
				return BackendVMware, nil
			}
			if findVBoxManage() != "" {
				return BackendVirtualBox, nil
			}
			return "", fmt.Errorf("no supported Vagrant backend detected on macOS ARM64\n\nInstall VMware Fusion: brew install --cask vmware-fusion\n  then: vagrant plugin install vagrant-vmware-desktop\nOr install VirtualBox from https://www.virtualbox.org/wiki/Downloads")
		}
		// Intel Mac — prefer VirtualBox.
		if findVBoxManage() != "" {
			return BackendVirtualBox, nil
		}
		if _, err := exec.LookPath("vmrun"); err == nil && vagrantPluginInstalled("vagrant-vmware-desktop") {
			return BackendVMware, nil
		}
		return "", fmt.Errorf("no supported Vagrant backend detected on macOS\n\nInstall VirtualBox: brew install --cask virtualbox\nOr install VMware Fusion: brew install --cask vmware-fusion\n  then: vagrant plugin install vagrant-vmware-desktop")

	case "linux":
		if _, err := exec.LookPath("virsh"); err == nil && vagrantPluginInstalled("vagrant-libvirt") {
			return BackendLibvirt, nil
		}
		if findVBoxManage() != "" {
			return BackendVirtualBox, nil
		}
		return "", fmt.Errorf("no supported Vagrant backend detected on Linux\n\nInstall libvirt + vagrant-libvirt:\n  sudo apt install libvirt-daemon-system && vagrant plugin install vagrant-libvirt\nOr install VirtualBox from https://www.virtualbox.org/wiki/Downloads")

	default:
		return "", fmt.Errorf("unsupported OS for Vagrant backend auto-detection: %s", runtime.GOOS)
	}
}

// vagrantPluginInstalled checks whether a Vagrant plugin is present.
func vagrantPluginInstalled(pluginName string) bool {
	out, err := exec.Command("vagrant", "plugin", "list").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), pluginName+" ") {
			return true
		}
	}
	return false
}

// isWindowsAdmin checks whether the current process is running elevated.
func isWindowsAdmin() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	// "net session" succeeds only when running as administrator.
	err := exec.Command("cmd", "/C", "net", "session").Run()
	return err == nil
}
