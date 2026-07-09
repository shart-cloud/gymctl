package exam

import "testing"

func TestBuiltInIncludesCKS(t *testing.T) {
	def, ok := BuiltIn("cks")
	if !ok {
		t.Fatal("expected built-in cks exam definition")
	}
	if def.TrackPrefix != "cks" {
		t.Fatalf("TrackPrefix = %q, want cks", def.TrackPrefix)
	}
	if def.DefaultBackend != "vagrant" {
		t.Fatalf("DefaultBackend = %q, want vagrant", def.DefaultBackend)
	}
	if len(def.Domains) != 6 {
		t.Fatalf("len(Domains) = %d, want 6", len(def.Domains))
	}
	if def.DomainWeights["Supply Chain Security"] != 20 {
		t.Fatalf("Supply Chain Security weight = %d, want 20", def.DomainWeights["Supply Chain Security"])
	}
}
