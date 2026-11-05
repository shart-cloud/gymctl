package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// A child process cannot cd or export in its parent shell. The function
// printed by `gymctl shell-init` hands gymctl a temp file via
// GYMCTL_SHELL_DIRECTIVES, runs the real binary with the terminal attached
// (colors, prompts and spinners keep working), then sources whatever gymctl
// wrote there. gymctl only writes shell-quoted cd/export/unset lines.
const shellDirectivesEnv = "GYMCTL_SHELL_DIRECTIVES"

const shellInitScript = `# gymctl shell integration: lets gymctl cd into drill workspaces and export
# $PCAP / $GYM_WORKDIR / $GYM_EXERCISE in this shell.
gymctl() {
  local __gymctl_d __gymctl_rc
  __gymctl_d=$(mktemp "${TMPDIR:-/tmp}/gymctl-sh.XXXXXX" 2>/dev/null) || { command gymctl "$@"; return; }
  GYMCTL_SHELL_DIRECTIVES="$__gymctl_d" command gymctl "$@"
  __gymctl_rc=$?
  if [ -s "$__gymctl_d" ]; then . "$__gymctl_d"; fi
  rm -f "$__gymctl_d"
  return $__gymctl_rc
}
`

type shellDirectives struct {
	unset  []string
	export map[string]string
	cd     string
}

// emit writes the directives for the shell function. It is a no-op when
// gymctl is not running under the shell integration.
func (d shellDirectives) emit() error {
	path := strings.TrimSpace(os.Getenv(shellDirectivesEnv))
	if path == "" {
		return nil
	}
	var b strings.Builder
	if len(d.unset) > 0 {
		fmt.Fprintf(&b, "unset %s\n", strings.Join(d.unset, " "))
	}
	for _, key := range sortedEnvKeys(d.export) {
		fmt.Fprintf(&b, "export %s=%s\n", key, shellQuoteValue(d.export[key]))
	}
	if d.cd != "" {
		fmt.Fprintf(&b, "cd %s\n", shellQuoteValue(d.cd))
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func shellIntegrationActive() bool {
	return strings.TrimSpace(os.Getenv(shellDirectivesEnv)) != ""
}

func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-init [bash|zsh]",
		Short: "Print shell integration (add: eval \"$(gymctl shell-init)\" to your rc file)",
		Long: `Print a shell function that lets gymctl change directory into a drill
workspace and export $PCAP, $GYM_WORKDIR and $GYM_EXERCISE in your shell.

Add this to ~/.bashrc or ~/.zshrc:

  eval "$(gymctl shell-init)"`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && args[0] != "bash" && args[0] != "zsh" {
				return fmt.Errorf("unsupported shell %q (use bash or zsh)", args[0])
			}
			fmt.Fprint(cmd.OutOrStdout(), shellInitScript)
			return nil
		},
	}
}
