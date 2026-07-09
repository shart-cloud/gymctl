package cli

import (
	"testing"

	internalexam "gymctl/internal/exam"
	"gymctl/internal/scenario"
)

func TestClassifyCKSDomainUsesTags(t *testing.T) {
	def, ok := internalexam.BuiltIn("cks")
	if !ok {
		t.Fatal("expected built-in cks exam definition")
	}

	ex := &scenario.Exercise{
		Spec: scenario.ExerciseSpec{
			Tags: []string{"cks", "cks-supply-chain-security"},
		},
	}

	got := classifyDomain(ex, def)
	want := "Supply Chain Security"
	if got != want {
		t.Fatalf("classifyDomain() = %q, want %q", got, want)
	}
}

func TestCKSBackendCompatibilityIsExact(t *testing.T) {
	def, ok := internalexam.BuiltIn("cks")
	if !ok {
		t.Fatal("expected built-in cks exam definition")
	}

	kindEntry := scenario.CatalogEntry{Exercise: &scenario.Exercise{Spec: scenario.ExerciseSpec{
		Environment: scenario.EnvironmentSpec{
			Type:       "kubernetes",
			Kubernetes: &scenario.KubernetesSpec{Provider: "kind"},
		},
	}}}
	vagrantEntry := scenario.CatalogEntry{Exercise: &scenario.Exercise{Spec: scenario.ExerciseSpec{
		Environment: scenario.EnvironmentSpec{
			Type:       "kubernetes",
			Kubernetes: &scenario.KubernetesSpec{Provider: "vagrant"},
		},
	}}}

	if !isBackendCompatible(kindEntry, def, "kind") {
		t.Fatal("expected kind CKS exercise to be kind-compatible")
	}
	if isBackendCompatible(kindEntry, def, "vagrant") {
		t.Fatal("expected kind CKS exercise not to be vagrant-compatible")
	}
	if !isBackendCompatible(vagrantEntry, def, "vagrant") {
		t.Fatal("expected vagrant CKS exercise to be vagrant-compatible")
	}
	if isBackendCompatible(vagrantEntry, def, "kind") {
		t.Fatal("expected vagrant CKS exercise not to be kind-compatible")
	}
}

func TestFilterScaffoldEntries(t *testing.T) {
	entries := []scenario.CatalogEntry{
		{Exercise: &scenario.Exercise{Spec: scenario.ExerciseSpec{ImplementationStatus: "scaffold"}}},
		{Exercise: &scenario.Exercise{Spec: scenario.ExerciseSpec{ImplementationStatus: "ready"}}},
	}

	filtered := filterScaffoldEntries(entries, false)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}

	filtered = filterScaffoldEntries(entries, true)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered include scaffold) = %d, want 2", len(filtered))
	}
}
