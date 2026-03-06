package cli

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"gymctl/internal/scenario"
)

const (
	ckaDomainTroubleshooting = "Troubleshooting"
	ckaDomainClusterArch     = "Cluster Architecture"
	ckaDomainServicesNet     = "Services and Networking"
	ckaDomainWorkloads       = "Workloads and Scheduling"
	ckaDomainStorage         = "Storage"
)

var ckaDomainOrder = []string{
	ckaDomainTroubleshooting,
	ckaDomainClusterArch,
	ckaDomainServicesNet,
	ckaDomainWorkloads,
	ckaDomainStorage,
}

var ckaDomainWeights = map[string]int{
	ckaDomainTroubleshooting: 30,
	ckaDomainClusterArch:     25,
	ckaDomainServicesNet:     20,
	ckaDomainWorkloads:       15,
	ckaDomainStorage:         10,
}

type examOptions struct {
	count           int
	durationMinutes int
	seed            int64
	backend         string
	playlist        string
	ckaOnly         bool
	domainBalanced  bool
}

type examSession struct {
	Version         int      `yaml:"version"`
	Mode            string   `yaml:"mode"`
	Playlist        string   `yaml:"playlist"`
	Backend         string   `yaml:"backend"`
	DurationMinutes int      `yaml:"durationMinutes"`
	Seed            int64    `yaml:"seed"`
	CKAOnly         bool     `yaml:"ckaOnly"`
	DomainBalanced  bool     `yaml:"domainBalanced"`
	GeneratedAt     string   `yaml:"generatedAt"`
	Exercises       []string `yaml:"exercises"`
}

func newExamCmd() *cobra.Command {
	opts := &examOptions{}

	cmd := &cobra.Command{
		Use:   "exam",
		Short: "Generate exam playlists",
	}

	ckaCmd := &cobra.Command{
		Use:   "cka",
		Short: "Generate a timed CKA-style exam run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.durationMinutes <= 0 {
				return fmt.Errorf("duration must be > 0 minutes")
			}
			if opts.count < 0 {
				return fmt.Errorf("count cannot be negative")
			}

			backend := strings.ToLower(strings.TrimSpace(opts.backend))
			switch backend {
			case "kind", "vagrant":
			default:
				return fmt.Errorf("unsupported backend %q (use kind or vagrant)", opts.backend)
			}

			entries, err := scenario.LoadCatalog(tasksDir)
			if err != nil {
				return err
			}

			candidates := filterExamCandidates(entries, opts.playlist, opts.ckaOnly, backend)
			if len(candidates) == 0 {
				return fmt.Errorf("no kubernetes exercises found for playlist %q with current filters", opts.playlist)
			}

			seed := opts.seed
			if seed == 0 {
				seed = time.Now().UnixNano()
			}

			selected := selectExamExercises(candidates, opts.count, opts.durationMinutes, seed, opts.domainBalanced)
			if len(selected) == 0 {
				return fmt.Errorf("failed to build exam run from available exercises")
			}

			names := make([]string, 0, len(selected))
			for _, entry := range selected {
				names = append(names, entry.Exercise.Metadata.Name)
			}

			session := examSession{
				Version:         1,
				Mode:            "cka",
				Playlist:        opts.playlist,
				Backend:         backend,
				DurationMinutes: opts.durationMinutes,
				Seed:            seed,
				CKAOnly:         opts.ckaOnly,
				DomainBalanced:  opts.domainBalanced,
				GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
				Exercises:       names,
			}

			sessionPath, err := writeExamSession(session)
			if err != nil {
				return err
			}

			printExamPlan(cmd, selected, sessionPath, session)
			return nil
		},
	}

	ckaCmd.Flags().IntVar(&opts.count, "count", 0, "Number of random tasks (0 = auto-fit by duration)")
	ckaCmd.Flags().IntVar(&opts.durationMinutes, "duration-minutes", 120, "Exam duration in minutes")
	ckaCmd.Flags().Int64Var(&opts.seed, "seed", 0, "Random seed for reproducible task selection")
	ckaCmd.Flags().StringVar(&opts.backend, "backend", "vagrant", "Kubernetes backend for this exam run (kind or vagrant)")
	ckaCmd.Flags().StringVar(&opts.playlist, "playlist", "k8s", "Playlist to sample from (currently: k8s)")
	ckaCmd.Flags().BoolVar(&opts.ckaOnly, "cka-only", true, "Exclude non-CKA tracks (GitOps/CI/pipeline-focused tracks)")
	ckaCmd.Flags().BoolVar(&opts.domainBalanced, "domain-balanced", true, "Bias task selection to CKA domain spread")

	cmd.AddCommand(ckaCmd)
	return cmd
}

func filterExamCandidates(entries []scenario.CatalogEntry, playlist string, ckaOnly bool, backend string) []scenario.CatalogEntry {
	playlist = strings.ToLower(strings.TrimSpace(playlist))
	if playlist == "" {
		playlist = "k8s"
	}

	out := make([]scenario.CatalogEntry, 0, len(entries))
	for _, entry := range entries {
		ex := entry.Exercise
		if ex == nil || ex.Spec.Environment.Type != "kubernetes" {
			continue
		}

		if !matchesPlaylist(ex, playlist) {
			continue
		}

		if ckaOnly && !isCKAAlignedTrack(ex.Metadata.Track) {
			continue
		}

		if !isBackendCompatible(entry, backend) {
			continue
		}

		out = append(out, entry)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Exercise.Metadata.Name < out[j].Exercise.Metadata.Name
	})

	return out
}

func matchesPlaylist(ex *scenario.Exercise, playlist string) bool {
	switch playlist {
	case "k8s", "kubernetes":
		return strings.HasPrefix(strings.ToLower(ex.Metadata.Track), "k8s")
	default:
		return false
	}
}

func isCKAAlignedTrack(track string) bool {
	track = strings.ToLower(strings.TrimSpace(track))
	allowed := map[string]bool{
		"k8s-fundamentals":    true,
		"k8s-networking":      true,
		"k8s-troubleshooting": true,
		"k8s-admin":           true,
		"k8s-storage":         true,
		"k8s-workloads":       true,
		"k8s-autoscaling":     true,
	}
	return allowed[track]
}

func isBackendCompatible(entry scenario.CatalogEntry, backend string) bool {
	backend = strings.ToLower(strings.TrimSpace(backend))
	if backend != "vagrant" {
		return true
	}

	ex := entry.Exercise
	if ex == nil || ex.Spec.Environment.Kubernetes == nil {
		return false
	}

	if strings.EqualFold(ex.Spec.Environment.Kubernetes.Provider, "vagrant") {
		return true
	}

	forbidden := []string{
		"docker exec",
		"kind create cluster",
		"kind delete cluster",
		"kind get",
		"kind-lab",
	}

	for _, check := range ex.Spec.Checks {
		if containsAnyLower(check.Script, forbidden) {
			return false
		}
	}
	for _, step := range ex.Spec.Environment.CustomSetup {
		if containsAnyLower(step.Script, forbidden) {
			return false
		}
	}

	return true
}

func containsAnyLower(text string, needles []string) bool {
	text = strings.ToLower(text)
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}

func selectExamExercises(candidates []scenario.CatalogEntry, count, durationMinutes int, seed int64, domainBalanced bool) []scenario.CatalogEntry {
	r := rand.New(rand.NewSource(seed))
	shuffled := append([]scenario.CatalogEntry(nil), candidates...)
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	if !domainBalanced {
		return selectSimple(shuffled, count, durationMinutes)
	}

	buckets := make(map[string][]scenario.CatalogEntry)
	for _, entry := range shuffled {
		domain := classifyCKADomain(entry.Exercise)
		buckets[domain] = append(buckets[domain], entry)
	}

	selected := make([]scenario.CatalogEntry, 0, 6)
	used := map[string]bool{}
	usedMinutes := 0

	if count > 0 {
		for _, domain := range ckaDomainOrder {
			if len(selected) >= count {
				break
			}
			entry, ok, rest := popFitting(buckets[domain], durationMinutes-usedMinutes, used)
			buckets[domain] = rest
			if !ok {
				continue
			}
			selected = append(selected, entry)
			used[entry.Exercise.Metadata.Name] = true
			usedMinutes += estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
		}

		for len(selected) < count {
			domain, ok := weightedDomainPick(r, buckets, durationMinutes-usedMinutes, used)
			if !ok {
				break
			}
			entry, ok, rest := popFitting(buckets[domain], durationMinutes-usedMinutes, used)
			buckets[domain] = rest
			if !ok {
				continue
			}
			selected = append(selected, entry)
			used[entry.Exercise.Metadata.Name] = true
			usedMinutes += estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
		}

		if len(selected) == 0 && len(shuffled) > 0 {
			selected = append(selected, shuffled[0])
		}
		return selected
	}

	for _, domain := range ckaDomainOrder {
		entry, ok, rest := popFitting(buckets[domain], durationMinutes-usedMinutes, used)
		buckets[domain] = rest
		if !ok {
			continue
		}
		selected = append(selected, entry)
		used[entry.Exercise.Metadata.Name] = true
		usedMinutes += estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
	}

	for {
		domain, ok := weightedDomainPick(r, buckets, durationMinutes-usedMinutes, used)
		if !ok {
			break
		}
		entry, ok, rest := popFitting(buckets[domain], durationMinutes-usedMinutes, used)
		buckets[domain] = rest
		if !ok {
			continue
		}
		selected = append(selected, entry)
		used[entry.Exercise.Metadata.Name] = true
		usedMinutes += estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
	}

	if len(selected) == 0 && len(shuffled) > 0 {
		selected = append(selected, shuffled[0])
	}

	return selected
}

func selectSimple(shuffled []scenario.CatalogEntry, count, durationMinutes int) []scenario.CatalogEntry {
	if count > 0 {
		if count > len(shuffled) {
			count = len(shuffled)
		}
		return shuffled[:count]
	}

	selected := make([]scenario.CatalogEntry, 0, 6)
	usedMinutes := 0
	for _, entry := range shuffled {
		taskMinutes := estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
		if usedMinutes+taskMinutes > durationMinutes {
			continue
		}
		selected = append(selected, entry)
		usedMinutes += taskMinutes
	}

	if len(selected) == 0 && len(shuffled) > 0 {
		selected = append(selected, shuffled[0])
	}
	return selected
}

func popFitting(bucket []scenario.CatalogEntry, remainingMinutes int, used map[string]bool) (scenario.CatalogEntry, bool, []scenario.CatalogEntry) {
	entries := append([]scenario.CatalogEntry(nil), bucket...)
	for i := 0; i < len(entries); i++ {
		name := entries[i].Exercise.Metadata.Name
		if used[name] {
			continue
		}
		mins := estimatedMinutes(entries[i].Exercise.Spec.EstimatedTime)
		if mins > remainingMinutes {
			continue
		}
		picked := entries[i]
		entries = append(entries[:i], entries[i+1:]...)
		return picked, true, entries
	}
	return scenario.CatalogEntry{}, false, entries
}

func weightedDomainPick(r *rand.Rand, buckets map[string][]scenario.CatalogEntry, remainingMinutes int, used map[string]bool) (string, bool) {
	total := 0
	available := make([]string, 0, len(ckaDomainOrder))
	for _, domain := range ckaDomainOrder {
		if hasFittingCandidate(buckets[domain], remainingMinutes, used) {
			available = append(available, domain)
			total += ckaDomainWeights[domain]
		}
	}

	if total == 0 {
		return "", false
	}

	pick := r.Intn(total)
	running := 0
	for _, domain := range available {
		running += ckaDomainWeights[domain]
		if pick < running {
			return domain, true
		}
	}

	return available[len(available)-1], true
}

func hasFittingCandidate(bucket []scenario.CatalogEntry, remainingMinutes int, used map[string]bool) bool {
	for _, entry := range bucket {
		if used[entry.Exercise.Metadata.Name] {
			continue
		}
		if estimatedMinutes(entry.Exercise.Spec.EstimatedTime) <= remainingMinutes {
			return true
		}
	}
	return false
}

func classifyCKADomain(ex *scenario.Exercise) string {
	if ex == nil {
		return ckaDomainTroubleshooting
	}

	tags := make([]string, 0, len(ex.Spec.Tags))
	for _, t := range ex.Spec.Tags {
		tags = append(tags, strings.ToLower(strings.TrimSpace(t)))
	}
	has := func(values ...string) bool {
		for _, t := range tags {
			for _, v := range values {
				if t == v {
					return true
				}
			}
		}
		return false
	}

	track := strings.ToLower(strings.TrimSpace(ex.Metadata.Track))
	switch track {
	case "k8s-admin":
		return ckaDomainClusterArch
	case "k8s-storage":
		return ckaDomainStorage
	case "k8s-networking":
		return ckaDomainServicesNet
	case "k8s-troubleshooting":
		return ckaDomainTroubleshooting
	case "k8s-workloads", "k8s-autoscaling":
		return ckaDomainWorkloads
	}

	if has("storage", "pvc", "pv", "storageclass", "volume", "statefulset", "access-mode", "reclaim") {
		return ckaDomainStorage
	}
	if has("networking", "service-discovery", "dns", "coredns", "networkpolicy", "ingress", "gateway", "service") {
		return ckaDomainServicesNet
	}
	if has("troubleshooting", "cluster-components", "nodes") {
		return ckaDomainTroubleshooting
	}
	if has("rbac", "etcd", "kubeadm", "cluster-admin", "upgrade", "psa", "crd", "operator", "kubeconfig") {
		return ckaDomainClusterArch
	}
	if has("resources", "scheduling", "workloads", "autoscaling", "hpa", "vpa", "jobs", "cronjob", "probes", "configmap") || track == "k8s-fundamentals" {
		return ckaDomainWorkloads
	}

	return ckaDomainTroubleshooting
}

func estimatedMinutes(raw string) int {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 25
	}

	if d, err := time.ParseDuration(raw); err == nil {
		mins := int(d.Minutes())
		if mins > 0 {
			return mins
		}
	}

	var n int
	if _, err := fmt.Sscanf(raw, "%dm", &n); err == nil && n > 0 {
		return n
	}
	if _, err := fmt.Sscanf(raw, "%dh", &n); err == nil && n > 0 {
		return n * 60
	}

	return 25
}

func writeExamSession(session examSession) (string, error) {
	gymDir, err := resolveGymDir()
	if err != nil {
		return "", err
	}

	examDir := filepath.Join(gymDir, "exams")
	if err := os.MkdirAll(examDir, 0o755); err != nil {
		return "", fmt.Errorf("create exams dir: %w", err)
	}

	id := fmt.Sprintf("%s-%d", session.Mode, time.Now().UTC().Unix())
	path := filepath.Join(examDir, id+".yaml")

	data, err := yaml.Marshal(&session)
	if err != nil {
		return "", fmt.Errorf("marshal exam session: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write exam session: %w", err)
	}

	return path, nil
}

func printExamPlan(cmd *cobra.Command, selected []scenario.CatalogEntry, sessionPath string, session examSession) {
	out := cmd.OutOrStdout()

	ColorHeader.Fprintln(out, "CKA Exam Mode")
	ColorDim.Fprintf(out, "Duration: %d minutes  |  Backend: %s  |  Seed: %d\n", session.DurationMinutes, session.Backend, session.Seed)
	ColorDim.Fprintf(out, "Filters: cka-only=%t, domain-balanced=%t\n", session.CKAOnly, session.DomainBalanced)
	fmt.Fprintln(out)

	totalMinutes := 0
	domainCounts := map[string]int{}
	for i, entry := range selected {
		mins := estimatedMinutes(entry.Exercise.Spec.EstimatedTime)
		totalMinutes += mins
		domain := classifyCKADomain(entry.Exercise)
		domainCounts[domain]++
		fmt.Fprintf(out, "%2d. %s (%s, %dm, %s)\n", i+1, entry.Exercise.Metadata.Name, entry.Exercise.Spec.Difficulty, mins, domain)
	}

	fmt.Fprintln(out)
	ColorInfo.Fprintf(out, "Estimated task time: %d minutes\n", totalMinutes)
	ColorInfo.Fprintf(out, "Session file: %s\n", sessionPath)
	fmt.Fprintln(out)

	ColorBold.Fprintln(out, "Domain spread:")
	for _, domain := range ckaDomainOrder {
		fmt.Fprintf(out, "- %s: %d\n", domain, domainCounts[domain])
	}
	fmt.Fprintln(out)

	ColorBold.Fprintln(out, "Runbook:")
	for i, entry := range selected {
		fmt.Fprintf(out, "%2d) gymctl start %s --provider %s\n", i+1, entry.Exercise.Metadata.Name, session.Backend)
		fmt.Fprintln(out, "   gymctl check --verbose --no-cleanup")
		fmt.Fprintln(out, "   gymctl stop")
	}
	fmt.Fprintln(out)
	ColorDim.Fprintln(out, "Tip: use --seed to reproduce this exact exam set.")
}
