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
	// ObjectiveAliases maps an exercise from another track onto this exam's
	// domains. Keys are "track:objective" (e.g. "sec503:4.1") or
	// "track:domain" (e.g. "sec503:1"); values are this exam's domain IDs.
	// The exact objective key wins over the domain key.
	ObjectiveAliases map[string]string
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

// AliasDomainID resolves a foreign-track exercise objective to one of this
// exam's domain IDs via ObjectiveAliases. ok is false when no alias applies.
func (d Definition) AliasDomainID(track, objective string) (string, bool) {
	if len(d.ObjectiveAliases) == 0 {
		return "", false
	}
	track = strings.ToLower(strings.TrimSpace(track))
	objective = strings.TrimSpace(objective)
	if id, ok := d.ObjectiveAliases[track+":"+objective]; ok {
		return id, true
	}
	domain := strings.Split(objective, ".")[0]
	if id, ok := d.ObjectiveAliases[track+":"+domain]; ok {
		return id, true
	}
	return "", false
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

	sec503Domains := []Domain{
		{ID: "1", Name: "Packets and Frames", Weight: 20},
		{ID: "2", Name: "Wireshark Deep Dive", Weight: 20},
		{ID: "3", Name: "IDS/IPS Rules", Weight: 15},
		{ID: "4", Name: "Traffic Analysis", Weight: 20},
		{ID: "5", Name: "Network Forensics", Weight: 20},
		{ID: "6", Name: "Advanced Topics", Weight: 5},
	}

	// GX-IA: the seven objective areas published on the GIAC exam page.
	// Weights are an even spread; GIAC does not publish per-domain weighting.
	gxIADomains := []Domain{
		{ID: "1", Name: "Advanced Analysis Techniques", Weight: 14},
		{ID: "2", Name: "Application Traffic Analysis", Weight: 14},
		{ID: "3", Name: "IDS Application and Analysis", Weight: 14},
		{ID: "4", Name: "Malicious Traffic Analysis", Weight: 15},
		{ID: "5", Name: "Network Forensics", Weight: 15},
		{ID: "6", Name: "Network Traffic Analysis", Weight: 14},
		{ID: "7", Name: "Protocol Analysis", Weight: 14},
	}

	// sec503 drills are strong GX-IA prep, so the gx-ia exam draws from both
	// tracks. sec503 objectives are numbered by course section, not GX-IA
	// domain; this crosswalk places each section (or specific objective) in
	// the GX-IA domain it exercises.
	gxIAAliases := map[string]string{
		"sec503:1":   "7", // Packets and Frames -> Protocol Analysis
		"sec503:2":   "2", // Wireshark Deep Dive -> Application Traffic Analysis
		"sec503:2.1": "1", // display vs capture filters -> Advanced Analysis Techniques
		"sec503:2.2": "1",
		"sec503:2.9": "1", // IO graphs
		"sec503:3":   "3", // Suricata rules -> IDS Application and Analysis
		"sec503:4":   "4", // Traffic Analysis -> Malicious Traffic Analysis
		"sec503:4.1": "6", // recon / scan stimulus-response -> Network Traffic Analysis
		"sec503:5":   "5", // Network Forensics
		"sec503:6":   "7", // IPv6 / VLAN -> Protocol Analysis
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
			ID:              "sec503",
			Name:            "SEC503 Network Monitoring & Threat Detection",
			Domains:         sec503Domains,
			DomainWeights:   weightsByName(sec503Domains),
			DomainOrder:     namesInOrder(sec503Domains),
			TrackPrefix:     "sec503",
			DefaultDuration: 120,
			DefaultBackend:  "local",
		},
		{
			ID:               "gx-ia",
			Name:             "GIAC Experienced Intrusion Analyst",
			Domains:          gxIADomains,
			DomainWeights:    weightsByName(gxIADomains),
			DomainOrder:      namesInOrder(gxIADomains),
			AllowedTracks:    []string{"gx-ia", "sec503"},
			TrackPrefix:      "gx-ia",
			ObjectiveAliases: gxIAAliases,
			DefaultDuration:  240,
			DefaultBackend:   "local",
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
