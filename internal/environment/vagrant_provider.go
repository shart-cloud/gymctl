package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gymctl/internal/runner"
	"gymctl/internal/scenario"
)

type VagrantProvider struct{}

func (p *VagrantProvider) Name() string {
	return KubernetesProviderVagrant
}

// checkVagrantDeps verifies that vagrant and VBoxManage are in PATH and
// returns an actionable error message if either is missing.
func checkVagrantDeps() error {
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

	// VBoxManage is the VirtualBox management CLI; it ships with VirtualBox.
	vboxCmd := "VBoxManage"
	if runtime.GOOS == "windows" {
		vboxCmd = "VBoxManage.exe"
	}
	if _, err := exec.LookPath(vboxCmd); err != nil {
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
		return fmt.Errorf("VBoxManage not found in PATH — is VirtualBox installed?\n\n%s", hint)
	}

	return nil
}

func (p *VagrantProvider) Ensure(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error {
	if err := checkVagrantDeps(); err != nil {
		return err
	}

	if state == nil {
		return fmt.Errorf("exercise state is required")
	}
	if state.ExerciseName == "" {
		return fmt.Errorf("exerciseName is required for vagrant provider")
	}

	state.Provider = p.Name()

	workspace, err := resolveVagrantWorkspace(state.ExerciseName)
	if err != nil {
		return err
	}
	state.WorkspaceDir = workspace

	machines := buildVagrantMachines(state.ExerciseName, spec)
	if len(machines) == 0 {
		return fmt.Errorf("vagrant topology resolved to zero machines")
	}

	state.Nodes = make(map[string]string, len(machines))
	for _, m := range machines {
		state.Nodes[m.LogicalName] = m.VMName
	}

	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return fmt.Errorf("create vagrant workspace: %w", err)
	}

	vagrantfile := buildVagrantfile(spec, machines)
	vagrantfilePath := filepath.Join(workspace, "Vagrantfile")
	if err := os.WriteFile(vagrantfilePath, []byte(vagrantfile), 0o644); err != nil {
		return fmt.Errorf("write Vagrantfile: %w", err)
	}

	if !ShouldCreateKubernetesCluster(spec) {
		return nil
	}

	if err := runner.RunStreamingInDir(ctx, workspace, "vagrant", "up"); err != nil {
		return err
	}

	return nil
}

func (p *VagrantProvider) Teardown(ctx context.Context, state *ExerciseState) error {
	workspace, err := vagrantWorkspaceFromState(state)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(workspace); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil
		}
		return statErr
	}

	if err := runner.RunStreamingInDir(ctx, workspace, "vagrant", "destroy", "-f"); err != nil {
		return err
	}

	return nil
}

func (p *VagrantProvider) Reset(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error {
	if err := p.Teardown(ctx, state); err != nil {
		return err
	}
	return p.Ensure(ctx, entryDir, spec, state)
}

func (p *VagrantProvider) NodeExec(ctx context.Context, state *ExerciseState, nodeRef string, script string, timeout string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("script is required")
	}

	workspace, err := vagrantWorkspaceFromState(state)
	if err != nil {
		return "", err
	}

	vmName, err := resolveVagrantNodeName(state, nodeRef)
	if err != nil {
		return "", err
	}

	runCtx := ctx
	cancel := func() {}
	if strings.TrimSpace(timeout) != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return "", fmt.Errorf("invalid timeout %q: %w", timeout, err)
		}
		runCtx, cancel = context.WithTimeout(ctx, d)
	}
	defer cancel()

	output, err := runner.RunInDir(runCtx, workspace, "vagrant", "ssh", vmName, "-c", script)
	if err != nil {
		return output, err
	}

	return output, nil
}

func (p *VagrantProvider) ExportKubeconfig(ctx context.Context, state *ExerciseState, destPath string) (string, error) {
	if state == nil {
		return "", fmt.Errorf("exercise state is required")
	}

	adminConf, err := p.NodeExec(ctx, state, "cp-1", "sudo cat /etc/kubernetes/admin.conf", "30s")
	if err != nil {
		return "", err
	}
	adminConf = strings.TrimSpace(adminConf)
	if adminConf == "" {
		return "", fmt.Errorf("empty kubeconfig read from cp-1")
	}

	controlPlaneIP, _ := p.NodeExec(ctx, state, "cp-1", "hostname -I | awk '{print $1}'", "10s")
	controlPlaneIP = strings.TrimSpace(controlPlaneIP)
	if controlPlaneIP != "" {
		adminConf = rewriteKubeconfigServer(adminConf, controlPlaneIP)
	}

	if strings.TrimSpace(destPath) == "" {
		var pathErr error
		destPath, pathErr = resolveExerciseKubeconfigPath(state.ExerciseName)
		if pathErr != nil {
			return "", pathErr
		}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("create kubeconfig directory: %w", err)
	}
	if err := os.WriteFile(destPath, []byte(adminConf+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}

	state.Kubeconfig = destPath
	return destPath, nil
}

func resolveVagrantWorkspace(exerciseName string) (string, error) {
	if exerciseName == "" {
		return "", fmt.Errorf("exerciseName is required")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".gym", "providers", "vagrant", exerciseName), nil
}

func vagrantWorkspaceFromState(state *ExerciseState) (string, error) {
	if state == nil {
		return "", fmt.Errorf("exercise state is required")
	}
	if strings.TrimSpace(state.WorkspaceDir) != "" {
		return state.WorkspaceDir, nil
	}
	if strings.TrimSpace(state.ExerciseName) == "" {
		return "", fmt.Errorf("exerciseName is required in state")
	}
	return resolveVagrantWorkspace(state.ExerciseName)
}

func resolveVagrantNodeName(state *ExerciseState, nodeRef string) (string, error) {
	if state == nil {
		return "", fmt.Errorf("exercise state is required")
	}
	nodeRef = strings.TrimSpace(nodeRef)
	if nodeRef == "" {
		return "", fmt.Errorf("node reference is required")
	}

	if runtimeName, ok := state.Nodes[nodeRef]; ok && runtimeName != "" {
		return runtimeName, nil
	}

	if strings.HasPrefix(nodeRef, "cp-") || strings.HasPrefix(nodeRef, "worker-") {
		if state.ExerciseName == "" {
			return "", fmt.Errorf("cannot derive vm name for node %q without exerciseName", nodeRef)
		}
		return sanitizeVMName(state.ExerciseName + "-" + nodeRef), nil
	}

	return nodeRef, nil
}

func buildVagrantMachines(exerciseName string, spec *scenario.KubernetesSpec) []vagrantMachine {
	controlPlanes := 1
	workers := 1

	cpCPUs := 2
	cpMemoryMB := 4096
	workerCPUs := 2
	workerMemoryMB := 3072

	ipPrefix := "192.168.56"
	ipStart := 10

	if spec != nil && spec.Vagrant != nil {
		if spec.Vagrant.Topology.ControlPlanes > 0 {
			controlPlanes = spec.Vagrant.Topology.ControlPlanes
		}
		if spec.Vagrant.Topology.Workers >= 0 {
			workers = spec.Vagrant.Topology.Workers
		}
		if spec.Vagrant.Resources.ControlPlane.CPUs > 0 {
			cpCPUs = spec.Vagrant.Resources.ControlPlane.CPUs
		}
		if spec.Vagrant.Resources.ControlPlane.MemoryMB > 0 {
			cpMemoryMB = spec.Vagrant.Resources.ControlPlane.MemoryMB
		}
		if spec.Vagrant.Resources.Worker.CPUs > 0 {
			workerCPUs = spec.Vagrant.Resources.Worker.CPUs
		}
		if spec.Vagrant.Resources.Worker.MemoryMB > 0 {
			workerMemoryMB = spec.Vagrant.Resources.Worker.MemoryMB
		}
		if prefix, ok := cidrPrefix(spec.Vagrant.Network.PrivateCIDR); ok {
			ipPrefix = prefix
		}
	}

	total := controlPlanes + workers
	machines := make([]vagrantMachine, 0, total)

	nextIP := ipStart
	for i := 1; i <= controlPlanes; i++ {
		logical := "cp-" + strconv.Itoa(i)
		vmName := sanitizeVMName(exerciseName + "-" + logical)
		ip := ""
		if ipPrefix != "" {
			ip = fmt.Sprintf("%s.%d", ipPrefix, nextIP)
			nextIP++
		}
		machines = append(machines, vagrantMachine{
			LogicalName: logical,
			VMName:      vmName,
			Role:        "control-plane",
			CPUs:        cpCPUs,
			MemoryMB:    cpMemoryMB,
			IP:          ip,
		})
	}

	for i := 1; i <= workers; i++ {
		logical := "worker-" + strconv.Itoa(i)
		vmName := sanitizeVMName(exerciseName + "-" + logical)
		ip := ""
		if ipPrefix != "" {
			ip = fmt.Sprintf("%s.%d", ipPrefix, nextIP)
			nextIP++
		}
		machines = append(machines, vagrantMachine{
			LogicalName: logical,
			VMName:      vmName,
			Role:        "worker",
			CPUs:        workerCPUs,
			MemoryMB:    workerMemoryMB,
			IP:          ip,
		})
	}

	return machines
}

func cidrPrefix(cidr string) (string, bool) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return "", false
	}

	ip := cidr
	if slash := strings.IndexByte(cidr, '/'); slash >= 0 {
		ip = cidr[:slash]
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "", false
	}
	return strings.Join(parts[:3], "."), true
}

func sanitizeVMName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	if name == "" {
		return "gym-node"
	}
	return name
}

func rewriteKubeconfigServer(content string, controlPlaneIP string) string {
	if strings.TrimSpace(controlPlaneIP) == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "server:") {
			prefix := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
			lines[i] = prefix + "server: https://" + controlPlaneIP + ":6443"
		}
	}
	return strings.Join(lines, "\n")
}
