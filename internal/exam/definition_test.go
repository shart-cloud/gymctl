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

func TestBuiltInGXIADrawsFromSEC503(t *testing.T) {
	def, ok := BuiltIn("gx-ia")
	if !ok {
		t.Fatal("expected built-in gx-ia exam definition")
	}
	if len(def.Domains) != 7 {
		t.Fatalf("len(Domains) = %d, want 7", len(def.Domains))
	}
	if len(def.AllowedTracks) != 2 || def.AllowedTracks[1] != "sec503" {
		t.Fatalf("AllowedTracks = %v, want [gx-ia sec503]", def.AllowedTracks)
	}
	cases := map[[2]string]string{
		{"sec503", "1.3"}: "7", // section default
		{"sec503", "4.1"}: "6", // exact objective beats section
		{"sec503", "4.3"}: "4",
		{"SEC503", "2.1"}: "1", // track match is case-insensitive
	}
	for in, want := range cases {
		got, ok := def.AliasDomainID(in[0], in[1])
		if !ok || got != want {
			t.Errorf("AliasDomainID(%q, %q) = %q,%v want %q", in[0], in[1], got, ok, want)
		}
	}
	if _, ok := def.AliasDomainID("gx-ia", "3.1"); ok {
		t.Error("gx-ia exercises must not be aliased")
	}
}
