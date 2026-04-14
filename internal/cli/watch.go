package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gymctl/internal/checks"
	"gymctl/internal/dialogue"
	"gymctl/internal/scenario"
)

type watchOptions struct {
	verbose  bool
	interval time.Duration
}

func newWatchCmd() *cobra.Command {
	opts := &watchOptions{}
	cmd := &cobra.Command{
		Use:   "watch [exercise-name]",
		Short: "Re-run checks automatically on file changes or a poll interval",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer RecoverFromPanic(cmd)

			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				current, err := loadCurrentExercise()
				if err != nil {
					return WrapErrorWithHint(
						fmt.Errorf("no exercise specified and no current exercise set"),
						"Start an exercise first or specify one",
						"gymctl start <exercise-name>",
					)
				}
				name = current
			}

			entries, err := loadCatalogEntries()
			if err != nil {
				return err
			}
			entry, found := scenario.FindByName(entries, name)
			if !found {
				return fmt.Errorf("exercise not found: %s", name)
			}

			baseCtx := cmd.Context()
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			if err := configureExerciseKubeconfigEnv(baseCtx, entry.Exercise); err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(baseCtx)
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				select {
				case <-sigCh:
					cancel()
				case <-ctx.Done():
				}
			}()

			switch entry.Exercise.Spec.Environment.Type {
			case "docker":
				workDir, err := resolveWorkDir(entry.Exercise.Metadata.Name)
				if err != nil {
					return err
				}
				return watchDocker(ctx, cmd, entry, workDir, opts)
			case "local":
				return watchDocker(ctx, cmd, entry, entry.Dir, opts)
			default:
				return watchKubernetes(ctx, cmd, entry, opts)
			}
		},
	}

	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Show check details")
	cmd.Flags().DurationVar(&opts.interval, "interval", 5*time.Second, "Poll interval for Kubernetes exercises")

	return cmd
}

func watchDocker(ctx context.Context, cmd *cobra.Command, entry *scenario.CatalogEntry, workDir string, opts *watchOptions) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	defer watcher.Close()

	// Add workDir and all existing subdirectories (fsnotify doesn't recurse automatically)
	if err := addDirRecursive(watcher, workDir); err != nil {
		return fmt.Errorf("watch directory %s: %w", workDir, err)
	}

	// Run initial check before waiting for events
	prevPassCount := -1
	done, prevPassCount := runChecksAndRender(ctx, cmd, entry, workDir, opts, prevPassCount)
	if done {
		return nil
	}

	// timerCh receives a signal after the debounce delay
	timerCh := make(chan struct{}, 1)
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				// If a new directory was created, watch it too
				if event.Has(fsnotify.Create) {
					if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
						_ = addDirRecursive(watcher, event.Name)
					}
				}
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
					if ctx.Err() != nil {
						return
					}
					select {
					case timerCh <- struct{}{}:
					default:
					}
				})
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			ColorDim.Fprintln(cmd.OutOrStdout(), "  watcher error")
		case <-timerCh:
			var done bool
			done, prevPassCount = runChecksAndRender(ctx, cmd, entry, workDir, opts, prevPassCount)
			if done {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func watchKubernetes(ctx context.Context, cmd *cobra.Command, entry *scenario.CatalogEntry, opts *watchOptions) error {
	// Run initial check
	prevPassCount := -1
	done, prevPassCount := runChecksAndRender(ctx, cmd, entry, "", opts, prevPassCount)
	if done {
		return nil
	}

	ticker := time.NewTicker(opts.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var done bool
			done, prevPassCount = runChecksAndRender(ctx, cmd, entry, "", opts, prevPassCount)
			if done {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func runChecksAndRender(ctx context.Context, cmd *cobra.Command, entry *scenario.CatalogEntry, workDir string, opts *watchOptions, prevPassCount int) (done bool, passCount int) {
	results, allPassed := checks.RunExerciseChecks(ctx, entry.Exercise, workDir)

	passedCount := 0
	for _, r := range results {
		if r.Passed {
			passedCount++
		}
	}

	out := cmd.OutOrStdout()
	clearScreen(out)
	renderWatchFrame(out, entry, results, allPassed, time.Now(), workDir, opts)

	if passedCount > prevPassCount && !allPassed && prevPassCount >= 0 {
		dialogue.RenderJerry(out, dialogue.Jerry(entry.Exercise.Spec.JerryDialog, "onCheckPass", entry.Exercise.Metadata.Name))
		fmt.Fprintln(out)
	}

	if allPassed {
		if err := markCompleted(entry.Exercise); err != nil {
			ColorDim.Fprintf(out, "  warning: could not save progress: %v\n", err)
		}
		dialogue.RenderJerry(out, dialogue.Jerry(entry.Exercise.Spec.JerryDialog, "onComplete", entry.Exercise.Metadata.Name))
		fmt.Fprintln(out)
		ColorSuccess.Fprintln(out, "Exercise complete! GRIM-9 has closed the ticket.")
		cleanupConfig := &CleanupConfig{
			AutoClean:       false,
			SkipClean:       false,
			CleanImages:     true,
			CleanContainers: true,
			CleanVolumes:    true,
			Exercise:        entry.Exercise.Metadata.Name,
		}
		CleanupHook(cmd, entry.Exercise, cleanupConfig)
		return true, passedCount
	}
	return false, passedCount
}

func renderWatchFrame(w io.Writer, entry *scenario.CatalogEntry, results []checks.Result, allPassed bool, lastChecked time.Time, watchDir string, opts *watchOptions) {
	ColorHeader.Fprintf(w, "WATCH · %s\n", entry.Exercise.Metadata.Title)

	statusLine := fmt.Sprintf("Last checked: %s", lastChecked.Format("15:04:05"))
	if watchDir != "" {
		statusLine += fmt.Sprintf("  ·  watching %s", watchDir)
	}
	ColorDim.Fprintln(w, statusLine)
	fmt.Fprintln(w)

	passedCount := 0
	for _, r := range results {
		if r.Passed {
			passedCount++
		}
	}
	fmt.Fprintln(w, ProgressBar(passedCount, len(results), 20))
	fmt.Fprintln(w)

	for _, r := range results {
		msg := ""
		if opts.verbose && r.Message != "" {
			msg = r.Message
		}
		fmt.Fprintf(w, "  %s\n", FormatCheckResult(r.Name, r.Passed, msg))
	}
	fmt.Fprintln(w)

	if !allPassed {
		dialogue.RenderJerry(w, dialogue.Jerry(entry.Exercise.Spec.JerryDialog, "onCheckFail", entry.Exercise.Metadata.Name))
		fmt.Fprintln(w)
		ColorDim.Fprintln(w, "  Press Ctrl+C to stop watching.")
	}
}

func clearScreen(w io.Writer) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprint(w, "\033[2J\033[H")
	}
}

// addDirRecursive adds a directory and all its subdirectories to the watcher.
func addDirRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}
