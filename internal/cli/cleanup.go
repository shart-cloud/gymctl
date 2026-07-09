package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

// CleanupConfig defines cleanup behavior
type CleanupConfig struct {
	AutoClean       bool   // Automatically clean without prompting
	SkipClean       bool   // Skip cleanup entirely
	CleanImages     bool   // Clean Docker images
	CleanVolumes    bool   // Clean Docker volumes
	CleanContainers bool   // Clean stopped containers
	ReportEmpty     bool   // Report when no matching artifacts are found
	Exercise        string // Exercise name for targeted cleanup
}

// CleanupHook runs cleanup after successful exercise completion
func CleanupHook(cmd *cobra.Command, exercise *scenario.Exercise, config *CleanupConfig) error {
	// Don't cleanup if skipped or not a Docker exercise
	if config.SkipClean || exercise.Spec.Environment.Type != "docker" {
		return nil
	}

	// Check if there are Docker artifacts to clean
	artifacts, err := detectDockerArtifacts(cmd.Context(), exercise)
	if err != nil {
		ColorDim.Fprintf(cmd.OutOrStdout(), "Could not detect Docker artifacts: %v\n", err)
		return nil
	}

	if artifacts.IsEmpty() {
		if config.ReportEmpty {
			ColorDim.Fprintln(cmd.OutOrStdout(), "No gymctl-attributed Docker artifacts found for this exercise.")
		}
		return nil
	}

	// Show what will be cleaned
	fmt.Fprintln(cmd.OutOrStdout())
	ColorHeader.Fprintln(cmd.OutOrStdout(), "🧹 Cleanup Available")

	if artifacts.ImageCount > 0 {
		ColorInfo.Fprintf(cmd.OutOrStdout(), "  • %d Docker image(s) - %s\n",
			artifacts.ImageCount, humanizeBytes(artifacts.ImageSize))
	}
	if artifacts.ContainerCount > 0 {
		ColorInfo.Fprintf(cmd.OutOrStdout(), "  • %d stopped container(s)\n", artifacts.ContainerCount)
	}
	if artifacts.VolumeCount > 0 {
		ColorInfo.Fprintf(cmd.OutOrStdout(), "  • %d volume(s) - %s\n",
			artifacts.VolumeCount, humanizeBytes(artifacts.VolumeSize))
	}

	totalSize := artifacts.ImageSize + artifacts.VolumeSize
	if totalSize > 0 {
		ColorBold.Fprintf(cmd.OutOrStdout(), "  Total space to reclaim: %s\n", humanizeBytes(totalSize))
	}

	// Decide whether to clean
	shouldClean := config.AutoClean
	if !shouldClean && !config.SkipClean {
		shouldClean = confirmAction(cmd, "\nClean up Docker artifacts?")
	}

	if !shouldClean {
		ColorDim.Fprintln(cmd.OutOrStdout(), "Skipping cleanup. Run 'gymctl cleanup' later to reclaim space.")
		return nil
	}

	// Perform cleanup
	spinner := NewSpinnerManager()
	spinner.Start("Cleaning up Docker artifacts")

	cleanedSize := int64(0)
	errors := []string{}

	// Clean images
	if config.CleanImages && artifacts.ImageCount > 0 {
		size, err := cleanDockerImages(cmd.Context(), exercise)
		if err != nil {
			errors = append(errors, fmt.Sprintf("images: %v", err))
		} else {
			cleanedSize += size
		}
	}

	// Clean containers
	if config.CleanContainers && artifacts.ContainerCount > 0 {
		err := cleanDockerContainers(cmd.Context(), exercise)
		if err != nil {
			errors = append(errors, fmt.Sprintf("containers: %v", err))
		}
	}

	// Clean volumes
	if config.CleanVolumes && artifacts.VolumeCount > 0 {
		size, err := cleanDockerVolumes(cmd.Context(), exercise)
		if err != nil {
			errors = append(errors, fmt.Sprintf("volumes: %v", err))
		} else {
			cleanedSize += size
		}
	}

	spinner.Stop()

	if len(errors) > 0 {
		ColorWarning.Fprintf(cmd.OutOrStdout(), "⚠ Cleanup completed with errors: %s\n", strings.Join(errors, ", "))
	} else if cleanedSize > 0 {
		ColorSuccess.Fprintf(cmd.OutOrStdout(), "✓ Cleaned up %s of Docker artifacts\n", humanizeBytes(cleanedSize))
	} else {
		ColorSuccess.Fprintln(cmd.OutOrStdout(), "✓ Cleanup complete")
	}

	return nil
}

// DockerArtifacts represents Docker resources that can be cleaned
type DockerArtifacts struct {
	ImageCount     int
	ImageSize      int64
	ContainerCount int
	VolumeCount    int
	VolumeSize     int64
	Images         []string
	Containers     []string
	Volumes        []string
}

func (a *DockerArtifacts) IsEmpty() bool {
	return a.ImageCount == 0 && a.ContainerCount == 0 && a.VolumeCount == 0
}

// detectDockerArtifacts finds Docker resources created by an exercise
func detectDockerArtifacts(ctx context.Context, exercise *scenario.Exercise) (*DockerArtifacts, error) {
	artifacts := &DockerArtifacts{}

	managedLabel := environment.DockerManagedLabel + "=true"
	exerciseLabel := environment.DockerExerciseLabel + "=" + exercise.Metadata.Name
	composeProjectLabel := "com.docker.compose.project=" + environment.DockerComposeProjectName(exercise.Metadata.Name)

	// Find images that gymctl built directly for this exercise.
	cmd := exec.CommandContext(ctx, "docker", "images", "--filter", "label="+managedLabel, "--filter", "label="+exerciseLabel, "--format", "{{.ID}}\t{{.Size}}")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				imageID := strings.TrimSpace(parts[0])
				if imageID == "" || contains(artifacts.Images, imageID) {
					continue
				}
				artifacts.Images = append(artifacts.Images, imageID)
				artifacts.ImageCount++
				artifacts.ImageSize += parseDockerSize(parts[1])
			}
		}
	}

	containerFilters := [][]string{
		{"label=" + managedLabel, "label=" + exerciseLabel},
		{"label=" + composeProjectLabel},
	}
	for _, filters := range containerFilters {
		args := []string{"ps", "-a", "--filter", "status=exited"}
		for _, filter := range filters {
			args = append(args, "--filter", filter)
		}
		args = append(args, "--format", "{{.ID}}")

		cmd = exec.CommandContext(ctx, "docker", args...)
		output, err = cmd.Output()
		if err == nil && len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				containerID := strings.TrimSpace(line)
				if containerID != "" && !contains(artifacts.Containers, containerID) {
					artifacts.Containers = append(artifacts.Containers, containerID)
					artifacts.ContainerCount++
				}
			}
		}
	}

	volumeFilters := [][]string{
		{"label=" + managedLabel, "label=" + exerciseLabel},
		{"label=" + composeProjectLabel},
	}
	for _, filters := range volumeFilters {
		args := []string{"volume", "ls"}
		for _, filter := range filters {
			args = append(args, "--filter", filter)
		}
		args = append(args, "--format", "{{.Name}}")

		cmd = exec.CommandContext(ctx, "docker", args...)
		output, err = cmd.Output()
		if err == nil && len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				volumeName := strings.TrimSpace(line)
				if volumeName == "" || contains(artifacts.Volumes, volumeName) {
					continue
				}
				artifacts.Volumes = append(artifacts.Volumes, volumeName)
				artifacts.VolumeCount++

				sizeCmd := exec.CommandContext(ctx, "docker", "volume", "inspect", volumeName, "--format", "{{.UsageData.Size}}")
				if sizeOutput, err := sizeCmd.Output(); err == nil {
					var size int64
					fmt.Sscanf(string(sizeOutput), "%d", &size)
					artifacts.VolumeSize += size
				}
			}
		}
	}

	return artifacts, nil
}

// cleanDockerImages removes Docker images for an exercise
func cleanDockerImages(ctx context.Context, exercise *scenario.Exercise) (int64, error) {
	artifacts, err := detectDockerArtifacts(ctx, exercise)
	if err != nil {
		return 0, err
	}

	totalSize := artifacts.ImageSize

	for _, imageID := range artifacts.Images {
		cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", imageID)
		cmd.Run() // Ignore errors for individual images
	}

	return totalSize, nil
}

// cleanDockerContainers removes stopped containers for an exercise
func cleanDockerContainers(ctx context.Context, exercise *scenario.Exercise) error {
	artifacts, err := detectDockerArtifacts(ctx, exercise)
	if err != nil {
		return err
	}

	for _, containerID := range artifacts.Containers {
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
		cmd.Run() // Ignore errors for individual containers
	}

	return nil
}

// cleanDockerVolumes removes volumes for an exercise
func cleanDockerVolumes(ctx context.Context, exercise *scenario.Exercise) (int64, error) {
	artifacts, err := detectDockerArtifacts(ctx, exercise)
	if err != nil {
		return 0, err
	}

	totalSize := artifacts.VolumeSize

	for _, volumeName := range artifacts.Volumes {
		cmd := exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volumeName)
		cmd.Run() // Ignore errors for individual volumes
	}

	return totalSize, nil
}

// parseDockerSize parses Docker's human-readable size format
func parseDockerSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	// Handle formats like "1.2GB", "500MB", "10.5KB"
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(sizeStr, suffix) {
			numStr := strings.TrimSuffix(sizeStr, suffix)
			var num float64
			fmt.Sscanf(numStr, "%f", &num)
			return int64(num * float64(multiplier))
		}
	}

	// Try to parse as raw number
	var size int64
	fmt.Sscanf(sizeStr, "%d", &size)
	return size
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// newCleanupCmd creates the cleanup command
func newCleanupCmd() *cobra.Command {
	var exerciseName string
	var all bool
	var force bool

	cmd := &cobra.Command{
		Use:   "cleanup [exercise-name]",
		Short: "Clean up Docker artifacts from exercises",
		Long: `Clean up Docker images, containers, and volumes created during exercises.
This helps reclaim disk space after completing exercises.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			if len(args) > 0 {
				exerciseName = args[0]
			}

			// If --all flag, do system-wide cleanup
			if all {
				return systemCleanup(cmd, ctx, force)
			}

			// If exercise specified, clean that exercise
			if exerciseName != "" {
				entries, err := loadCatalogEntries()
				if err != nil {
					return err
				}

				entry, found := scenario.FindByName(entries, exerciseName)
				if !found {
					return fmt.Errorf("exercise not found: %s", exerciseName)
				}

				config := &CleanupConfig{
					AutoClean:       force,
					CleanImages:     true,
					CleanContainers: true,
					CleanVolumes:    true,
					ReportEmpty:     true,
					Exercise:        exerciseName,
				}

				return CleanupHook(cmd, entry.Exercise, config)
			}

			// Otherwise show cleanup options
			return interactiveCleanup(cmd, ctx)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Clean all Docker artifacts (system-wide)")
	cmd.Flags().BoolVar(&force, "force", false, "Force cleanup without confirmation")

	return cmd
}

// systemCleanup performs system-wide Docker cleanup
func systemCleanup(cmd *cobra.Command, ctx context.Context, force bool) error {
	ColorHeader.Fprintln(cmd.OutOrStdout(), "🧹 System-wide Docker Cleanup")
	fmt.Fprintln(cmd.OutOrStdout())

	spinner := NewSpinnerManager()

	// Get current disk usage
	spinner.Start("Analyzing Docker disk usage")

	dfCmd := exec.CommandContext(ctx, "docker", "system", "df")
	dfOutput, _ := dfCmd.Output()

	spinner.Stop()

	if len(dfOutput) > 0 {
		ColorDim.Fprintln(cmd.OutOrStdout(), "Current Docker disk usage:")
		fmt.Fprintln(cmd.OutOrStdout(), string(dfOutput))
	}

	if !force && !confirmAction(cmd, "Proceed with system-wide cleanup?") {
		return nil
	}

	// Run Docker system prune
	spinner.Start("Running Docker system cleanup")

	pruneCmd := exec.CommandContext(ctx, "docker", "system", "prune", "-a", "-f", "--volumes")
	pruneOutput, err := pruneCmd.Output()

	spinner.Stop()

	if err != nil {
		ColorError.Fprintf(cmd.OutOrStdout(), "Cleanup failed: %v\n", err)
		return err
	}

	ColorSuccess.Fprintln(cmd.OutOrStdout(), "✓ System cleanup complete")
	if len(pruneOutput) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), string(pruneOutput))
	}

	return nil
}

// interactiveCleanup shows cleanup options interactively
func interactiveCleanup(cmd *cobra.Command, ctx context.Context) error {
	ColorHeader.Fprintln(cmd.OutOrStdout(), "🧹 Docker Cleanup Options")
	fmt.Fprintln(cmd.OutOrStdout())

	// List completed exercises with artifacts
	entries, err := loadCatalogEntries()
	if err != nil {
		return err
	}

	progressPath, _ := resolveProgressFile()
	progressFile, _ := progress.Load(progressPath)

	hasArtifacts := false
	for _, entry := range entries {
		exercise := entry.Exercise
		if exercise.Spec.Environment.Type != "docker" {
			continue
		}

		status := progressFile.Exercises[exercise.Metadata.Name]
		if status.Status == "completed" {
			artifacts, err := detectDockerArtifacts(ctx, exercise)
			if err == nil && !artifacts.IsEmpty() {
				if !hasArtifacts {
					ColorBold.Fprintln(cmd.OutOrStdout(), "Completed exercises with artifacts:")
					hasArtifacts = true
				}

				totalSize := artifacts.ImageSize + artifacts.VolumeSize
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s - %s\n",
					exercise.Metadata.Name,
					humanizeBytes(totalSize))
			}
		}
	}

	if !hasArtifacts {
		ColorInfo.Fprintln(cmd.OutOrStdout(), "No gymctl-attributed Docker artifacts found from completed exercises.")
		fmt.Fprintln(cmd.OutOrStdout())
		ColorDim.Fprintln(cmd.OutOrStdout(), "Older unlabeled resources are ignored. Use 'gymctl cleanup --all' for explicit system-wide Docker cleanup.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout())
	ColorInfo.Fprintln(cmd.OutOrStdout(), "To clean a specific exercise:")
	ColorDim.Fprintln(cmd.OutOrStdout(), "  gymctl cleanup <exercise-name>")
	fmt.Fprintln(cmd.OutOrStdout())
	ColorInfo.Fprintln(cmd.OutOrStdout(), "To clean all Docker artifacts:")
	ColorDim.Fprintln(cmd.OutOrStdout(), "  gymctl cleanup --all")

	return nil
}
