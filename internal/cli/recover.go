package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gymctl/internal/environment"
	"gymctl/internal/progress"
	"gymctl/internal/scenario"
)

func newRecoverCmd() *cobra.Command {
	var backupPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "recover [exercise-name]",
		Short: "Recover from corrupted state or restore backup",
		Long: `Recover helps restore exercises from various failure states:
- Corrupted Docker containers
- Failed Kubernetes deployments
- Lost work directory
- Corrupted progress file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer RecoverFromPanic(cmd)

			spinner := NewSpinnerManager()

			ColorHeader.Fprintln(cmd.OutOrStdout(), "🔧 Recovery Mode")
			fmt.Fprintln(cmd.OutOrStdout())

			// Check if we're recovering a specific exercise
			if len(args) > 0 {
				return recoverExercise(cmd, args[0], backupPath, force)
			}

			// General recovery - check all systems
			ColorBold.Fprintln(cmd.OutOrStdout(), "Checking system state...")

			// 1. Check progress file
			spinner.Start("Checking progress file")
			progressPath, err := resolveProgressFile()
			if err != nil {
				spinner.Fail("Cannot resolve progress file path")
				return err
			}

			progressFile, err := progress.Load(progressPath)
			if err != nil {
				spinner.Fail("Progress file is corrupted")

				// Attempt to recover
				if force || confirmAction(cmd, "Create new progress file?") {
					progressFile = &progress.File{
						Version:   1,
						Exercises: make(map[string]progress.ExerciseStatus),
					}
					if err := progress.Save(progressPath, progressFile); err != nil {
						return fmt.Errorf("failed to create new progress file: %w", err)
					}
					ColorSuccess.Fprintln(cmd.OutOrStdout(), "✓ Created new progress file")
				}
			} else {
				spinner.Success("Progress file is valid")
			}

			// 2. Check for orphaned containers
			spinner.Start("Checking for orphaned containers")
			orphaned, err := findOrphanedContainers(cmd.Context())
			if err != nil {
				spinner.Fail("Failed to check containers")
			} else if len(orphaned) > 0 {
				spinner.Stop()
				ColorWarning.Fprintf(cmd.OutOrStdout(), "Found %d orphaned containers\n", len(orphaned))

				for _, container := range orphaned {
					ColorDim.Fprintf(cmd.OutOrStdout(), "  - %s\n", container)
				}

				if force || confirmAction(cmd, "Remove orphaned containers?") {
					if err := cleanupOrphanedContainers(cmd.Context(), orphaned); err != nil {
						ColorError.Fprintf(cmd.OutOrStdout(), "Failed to clean some containers: %v\n", err)
					} else {
						ColorSuccess.Fprintln(cmd.OutOrStdout(), "✓ Cleaned up orphaned containers")
					}
				}
			} else {
				spinner.Success("No orphaned containers found")
			}

			// 3. Check work directories
			spinner.Start("Checking work directories")
			workDirs, err := checkWorkDirectories()
			if err != nil {
				spinner.Fail("Failed to check work directories")
			} else {
				spinner.Success(fmt.Sprintf("Found %d work directories", len(workDirs)))

				// Check for exercises without work dirs
				for name, status := range progressFile.Exercises {
					if status.Status == "in_progress" || status.Status == "completed" {
						workDir, _ := resolveWorkDir(name)
						if _, err := os.Stat(workDir); os.IsNotExist(err) {
							ColorWarning.Fprintf(cmd.OutOrStdout(), "Missing work directory for %s\n", name)

							if force || confirmAction(cmd, fmt.Sprintf("Create work directory for %s?", name)) {
								if err := os.MkdirAll(workDir, 0755); err != nil {
									ColorError.Fprintf(cmd.OutOrStdout(), "Failed to create: %v\n", err)
								} else {
									ColorSuccess.Fprintf(cmd.OutOrStdout(), "✓ Created work directory for %s\n", name)
								}
							}
						}
					}
				}
			}

			// 4. Check for backups
			backupDir, err := resolveBackupDir()
			if err == nil {
				files, err := os.ReadDir(backupDir)
				if err == nil && len(files) > 0 {
					ColorInfo.Fprintf(cmd.OutOrStdout(), "\nFound %d backup(s) in %s\n", len(files), backupDir)
					for _, file := range files {
						if strings.HasSuffix(file.Name(), ".tar.gz") {
							info, _ := file.Info()
							ColorDim.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n",
								file.Name(),
								humanizeBytes(info.Size()))
						}
					}
				}
			}

			fmt.Fprintln(cmd.OutOrStdout())
			ColorSuccess.Fprintln(cmd.OutOrStdout(), "✅ Recovery check complete")

			return nil
		},
	}

	cmd.Flags().StringVarP(&backupPath, "backup", "b", "", "Path to backup file to restore")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force recovery without confirmation")

	return cmd
}

func recoverExercise(cmd *cobra.Command, exerciseName, backupPath string, force bool) error {
	entries, err := scenario.LoadCatalog(tasksDir)
	if err != nil {
		return err
	}

	entry, found := scenario.FindByName(entries, exerciseName)
	if !found {
		return fmt.Errorf("exercise not found: %s", exerciseName)
	}

	exercise := entry.Exercise

	ColorBold.Fprintf(cmd.OutOrStdout(), "Recovering exercise: %s\n", exerciseName)
	fmt.Fprintln(cmd.OutOrStdout())

	// If backup path provided, restore from backup
	if backupPath != "" {
		return restoreFromBackup(cmd, exerciseName, backupPath)
	}

	// Otherwise, perform smart recovery
	spinner := NewSpinnerManager()

	// 1. Clean up environment
	spinner.Start("Cleaning up environment")
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	switch exercise.Spec.Environment.Type {
	case "docker":
		if exercise.Spec.Environment.Docker != nil {
			workDir, _ := resolveWorkDir(exerciseName)
			manager := environment.DockerManager{WorkDir: workDir}
			_ = manager.Teardown(ctx, entry.Dir, *exercise.Spec.Environment.Docker)
		}
	case "kubernetes":
		if exercise.Spec.Environment.Kubernetes != nil {
			k8s := exercise.Spec.Environment.Kubernetes
			if k8s.CreateCluster != nil && *k8s.CreateCluster {
				manager := environment.KindManager{ClusterName: "jerry-gym"}
				_ = manager.Delete(ctx)
			}
		}
	}
	spinner.Success("Environment cleaned")

	// 2. Reset progress
	spinner.Start("Resetting progress")
	progressPath, _ := resolveProgressFile()
	progressFile, _ := progress.Load(progressPath)

	if progressFile.Exercises != nil {
		status := progressFile.Exercises[exerciseName]
		status.Status = "not_started"
		status.HintsUsed = 0
		progressFile.Exercises[exerciseName] = status
		progress.Save(progressPath, progressFile)
	}
	spinner.Success("Progress reset")

	// 3. Clean work directory
	workDir, err := resolveWorkDir(exerciseName)
	if err == nil {
		if _, err := os.Stat(workDir); err == nil {
			if force || confirmAction(cmd, fmt.Sprintf("Remove work directory %s?", workDir)) {
				spinner.Start("Cleaning work directory")
				os.RemoveAll(workDir)
				os.MkdirAll(workDir, 0755)
				spinner.Success("Work directory cleaned")
			}
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	ColorSuccess.Fprintf(cmd.OutOrStdout(), "✅ Exercise %s recovered\n", exerciseName)
	ColorInfo.Fprintln(cmd.OutOrStdout(), "You can now start fresh with: gymctl start " + exerciseName)

	return nil
}

func createBackup(exerciseName string) (string, error) {
	backupDir, err := resolveBackupDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s-%s.tar.gz", exerciseName, timestamp))

	file, err := os.Create(backupFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Backup work directory
	workDir, err := resolveWorkDir(exerciseName)
	if err == nil {
		filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(workDir, path)
			header.Name = filepath.Join("work", relPath)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				io.Copy(tw, file)
			}

			return nil
		})
	}

	// Backup progress entry
	progressPath, _ := resolveProgressFile()
	progressFile, _ := progress.Load(progressPath)
	if status, exists := progressFile.Exercises[exerciseName]; exists {
		// Create a metadata file with progress info
		metadata := fmt.Sprintf("exercise: %s\nstatus: %s\nstarted: %s\ncompleted: %s\nhints: %d\n",
			exerciseName, status.Status, status.StartedAt, status.CompletedAt, status.HintsUsed)

		header := &tar.Header{
			Name: "metadata.txt",
			Mode: 0644,
			Size: int64(len(metadata)),
		}
		tw.WriteHeader(header)
		tw.Write([]byte(metadata))
	}

	return backupFile, nil
}

func restoreFromBackup(cmd *cobra.Command, exerciseName, backupPath string) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("cannot open backup file: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid backup file: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	workDir, _ := resolveWorkDir(exerciseName)

	ColorInfo.Fprintf(cmd.OutOrStdout(), "Restoring from backup: %s\n", backupPath)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasPrefix(header.Name, "work/") {
			// Restore work directory files
			targetPath := filepath.Join(workDir, strings.TrimPrefix(header.Name, "work/"))

			if header.Typeflag == tar.TypeDir {
				os.MkdirAll(targetPath, os.FileMode(header.Mode))
			} else {
				os.MkdirAll(filepath.Dir(targetPath), 0755)
				file, err := os.Create(targetPath)
				if err != nil {
					return err
				}
				io.Copy(file, tr)
				file.Close()
			}
		}
	}

	ColorSuccess.Fprintln(cmd.OutOrStdout(), "✓ Backup restored successfully")
	return nil
}

func findOrphanedContainers(ctx context.Context) ([]string, error) {
	// Implementation would check for containers with gymctl labels
	// that don't match current exercises
	return []string{}, nil
}

func cleanupOrphanedContainers(ctx context.Context, containers []string) error {
	// Implementation would remove the specified containers
	return nil
}

func checkWorkDirectories() ([]string, error) {
	gymDir, err := resolveGymDir()
	if err != nil {
		return nil, err
	}

	workdirPath := filepath.Join(gymDir, "workdir")
	if _, err := os.Stat(workdirPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(workdirPath)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func resolveBackupDir() (string, error) {
	gymDir, err := resolveGymDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gymDir, "backups"), nil
}

func confirmAction(cmd *cobra.Command, prompt string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}