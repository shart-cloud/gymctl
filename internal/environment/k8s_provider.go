package environment

import (
	"context"
	"fmt"
	"strings"

	"gymctl/internal/scenario"
)

const (
	KubernetesProviderKind    = "kind"
	KubernetesProviderVagrant = "vagrant"

	VagrantBootstrapManaged  = "managed"
	VagrantBootstrapBareNode = "barenode"
	// VagrantBootstrapPrebaked is for Packer-baked boxes that already ship
	// containerd + kubeadm/kubelet/kubectl + kernel/sysctl prep + pre-pulled
	// control-plane images. It skips the (slow) preinstall provisioner and only
	// runs the kubeadm init/join bootstrap.
	VagrantBootstrapPrebaked = "prebaked"
)

type KubernetesProvider interface {
	Ensure(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error
	Teardown(ctx context.Context, state *ExerciseState) error
	Reset(ctx context.Context, entryDir string, spec *scenario.KubernetesSpec, state *ExerciseState) error
	NodeExec(ctx context.Context, state *ExerciseState, nodeRef string, script string, timeout string) (string, error)
	ExportKubeconfig(ctx context.Context, state *ExerciseState, destPath string) (string, error)
	Name() string
}

func ResolveKubernetesProvider(spec *scenario.KubernetesSpec) (KubernetesProvider, error) {
	provider := KubernetesProviderKind
	if spec != nil && strings.TrimSpace(spec.Provider) != "" {
		provider = strings.ToLower(strings.TrimSpace(spec.Provider))
	}

	switch provider {
	case KubernetesProviderKind:
		return &KindProvider{}, nil
	case KubernetesProviderVagrant:
		return &VagrantProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported kubernetes provider: %s", provider)
	}
}

func ShouldCreateKubernetesCluster(spec *scenario.KubernetesSpec) bool {
	if spec == nil || spec.CreateCluster == nil {
		return true
	}
	return *spec.CreateCluster
}

func ResolveVagrantBootstrapMode(spec *scenario.KubernetesSpec) string {
	if spec == nil || spec.Vagrant == nil || strings.TrimSpace(spec.Vagrant.BootstrapMode) == "" {
		return VagrantBootstrapManaged
	}
	mode := strings.ToLower(strings.TrimSpace(spec.Vagrant.BootstrapMode))
	if mode != VagrantBootstrapManaged && mode != VagrantBootstrapBareNode && mode != VagrantBootstrapPrebaked {
		return VagrantBootstrapManaged
	}
	return mode
}

func IsBareNodeKubernetes(spec *scenario.KubernetesSpec) bool {
	provider := KubernetesProviderKind
	if spec != nil && strings.TrimSpace(spec.Provider) != "" {
		provider = strings.ToLower(strings.TrimSpace(spec.Provider))
	}
	return provider == KubernetesProviderVagrant && ResolveVagrantBootstrapMode(spec) == VagrantBootstrapBareNode
}
