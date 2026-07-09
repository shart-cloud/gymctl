package exam

import "strings"

type Definition struct {
	ID              string
	Name            string
	Domains         []Domain
	DomainWeights   map[string]int
	DomainOrder     []string
	TrackPrefix     string
	AllowedTracks   []string
	DefaultDuration int
	DefaultBackend  string
}

type Domain struct {
	ID     string
	Name   string
	Weight int
}

func BuiltIn(id string) (Definition, bool) {
	for _, def := range builtIns() {
		if strings.EqualFold(def.ID, id) {
			return def, true
		}
	}
	return Definition{}, false
}

func IDs() []string {
	defs := builtIns()
	ids := make([]string, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.ID)
	}
	return ids
}

func (d Definition) DomainNameByID(id string) string {
	for _, domain := range d.Domains {
		if domain.ID == id {
			return domain.Name
		}
	}
	if len(d.Domains) == 0 {
		return ""
	}
	return d.Domains[0].Name
}

func builtIns() []Definition {
	ckaDomains := []Domain{
		{ID: "troubleshooting", Name: "Troubleshooting", Weight: 30},
		{ID: "cluster-architecture", Name: "Cluster Architecture", Weight: 25},
		{ID: "services-networking", Name: "Services and Networking", Weight: 20},
		{ID: "workloads-scheduling", Name: "Workloads and Scheduling", Weight: 15},
		{ID: "storage", Name: "Storage", Weight: 10},
	}

	gxCSDomains := []Domain{
		{ID: "1", Name: "Advanced Network Analysis", Weight: 15},
		{ID: "2", Name: "Evaluating Linux Systems", Weight: 15},
		{ID: "3", Name: "Evaluating Windows Systems", Weight: 15},
		{ID: "4", Name: "File Analysis", Weight: 15},
		{ID: "5", Name: "Malicious Program Execution & Exploitation", Weight: 12},
		{ID: "6", Name: "Network Security", Weight: 16},
		{ID: "7", Name: "Password Cracking", Weight: 12},
	}

	cksDomains := []Domain{
		{ID: "cluster-setup", Name: "Cluster Setup", Weight: 15},
		{ID: "cluster-hardening", Name: "Cluster Hardening", Weight: 15},
		{ID: "system-hardening", Name: "System Hardening", Weight: 10},
		{ID: "minimize-microservice-vulnerabilities", Name: "Minimize Microservice Vulnerabilities", Weight: 20},
		{ID: "supply-chain-security", Name: "Supply Chain Security", Weight: 20},
		{ID: "monitoring-logging-runtime-security", Name: "Monitoring, Logging & Runtime Security", Weight: 20},
	}

	return []Definition{
		{
			ID:              "cka",
			Name:            "CKA Exam Mode",
			Domains:         ckaDomains,
			DomainWeights:   weightsByName(ckaDomains),
			DomainOrder:     namesInOrder(ckaDomains),
			AllowedTracks:   []string{"k8s-fundamentals", "k8s-networking", "k8s-troubleshooting", "k8s-admin", "k8s-storage", "k8s-workloads", "k8s-autoscaling"},
			TrackPrefix:     "k8s",
			DefaultDuration: 120,
			DefaultBackend:  "vagrant",
		},
		{
			ID:              "gx-cs",
			Name:            "GIAC Experienced Cyber Security",
			Domains:         gxCSDomains,
			DomainWeights:   weightsByName(gxCSDomains),
			DomainOrder:     namesInOrder(gxCSDomains),
			TrackPrefix:     "gx-cs",
			DefaultDuration: 240,
			DefaultBackend:  "local",
		},
		{
			ID:              "cks",
			Name:            "CKS Exam Mode",
			Domains:         cksDomains,
			DomainWeights:   weightsByName(cksDomains),
			DomainOrder:     namesInOrder(cksDomains),
			AllowedTracks:   []string{"cks"},
			TrackPrefix:     "cks",
			DefaultDuration: 120,
			DefaultBackend:  "vagrant",
		},
	}
}

func weightsByName(domains []Domain) map[string]int {
	weights := make(map[string]int, len(domains))
	for _, domain := range domains {
		weights[domain.Name] = domain.Weight
	}
	return weights
}

func namesInOrder(domains []Domain) []string {
	names := make([]string, 0, len(domains))
	for _, domain := range domains {
		names = append(names, domain.Name)
	}
	return names
}
