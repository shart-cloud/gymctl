package cloudlab

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const defaultWebhookURL = "https://labs.shart.cloud/api/labs/complete"

// RunPrompt prints the numbered exercise prompts from the loaded scenario.
// It is called on shell login (via .bashrc) and by `gymctl lab prompt`.
func RunPrompt(w io.Writer, scenarioPath string) error {
	scenario, err := Load(scenarioPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "\n┌─────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(w, "│  %-51s│\n", scenario.Meta.Title)
	fmt.Fprintf(w, "│  difficulty: %-37s│\n", scenario.Meta.Difficulty)
	fmt.Fprintf(w, "│  time limit: %-2d min                                  │\n", scenario.Meta.TimeLimitMinutes)
	fmt.Fprintf(w, "└─────────────────────────────────────────────────────┘\n")
	fmt.Fprintln(w)

	for i, c := range scenario.Checks {
		fmt.Fprintf(w, "  %d. %s\n", i+1, c.Prompt)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use kubectl to investigate, then run: gymctl lab check")
	fmt.Fprintln(w)

	return nil
}

// RunCheck runs the interactive check flow for the loaded scenario.
// For exact_match checks it prompts the student for an answer.
// For kubectl_output checks it runs the command and verifies stdout.
// On all checks passing it POSTs a signed completion to the webhook.
func RunCheck(scenarioPath string) error {
	scenario, err := Load(scenarioPath)
	if err != nil {
		return err
	}

	sessionID := os.Getenv("SESSION_ID")
	userID := os.Getenv("USER_ID")
	labID := os.Getenv("LAB_ID")
	webhookURL := os.Getenv("COMPLETION_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = defaultWebhookURL
	}

	if sessionID == "" || userID == "" || labID == "" {
		return fmt.Errorf("SESSION_ID, USER_ID, and LAB_ID env vars are required (set by container entrypoint)")
	}

	// Read the webhook secret from the setgid-protected file. The file is owned
	// root:labops mode 040, and gymctl is setgid labops, so labuser cannot read
	// it directly — `env`, `cat`, etc. won't expose the secret.
	// Fall back to the env var for local development (where the file won't exist).
	webhookSecret := ""
	if raw, err := os.ReadFile("/run/lab/completion_secret"); err == nil {
		webhookSecret = strings.TrimSpace(string(raw))
	} else {
		webhookSecret = os.Getenv("COMPLETION_WEBHOOK_SECRET")
	}

	scanner := bufio.NewScanner(os.Stdin)
	passed := 0
	total := len(scenario.Checks)

	fmt.Printf("\n=== Checking: %s ===\n\n", scenario.Meta.Title)

	for i := range scenario.Checks {
		c := &scenario.Checks[i]
		var ok bool

		switch c.Type {
		case "exact_match":
			ok, err = runExactMatch(scanner, i+1, c)
		case "kubectl_output":
			ok, err = runKubectlOutput(i+1, c)
		default:
			fmt.Printf("  [%d] Unknown check type %q — skipping\n\n", i+1, c.Type)
			continue
		}

		if err != nil {
			fmt.Printf("  [%d] Error: %v\n\n", i+1, err)
			continue
		}

		if ok {
			passed++
			fmt.Printf("  ✓ Correct!\n\n")
		} else {
			fmt.Printf("  ✗ Incorrect. Keep investigating.\n\n")
		}
	}

	fmt.Printf("Result: %d/%d checks passed\n\n", passed, total)

	if passed == total {
		fmt.Println("All checks passed!")
		if webhookSecret != "" {
			fmt.Println("Submitting completion...")
			if err := sendCompletion(webhookURL, webhookSecret, sessionID, userID, labID, passed, total); err != nil {
				fmt.Printf("Warning: could not submit completion: %v\n", err)
			} else {
				fmt.Println("Completion recorded. Well done!")
			}
		}
	} else {
		remaining := total - passed
		fmt.Printf("%d check(s) still need work. Run 'gymctl lab check' again when ready.\n", remaining)
	}

	return nil
}

// ─── check type implementations ──────────────────────────────────────────────

func runExactMatch(scanner *bufio.Scanner, num int, c *Check) (bool, error) {
	fmt.Printf("  [%d] %s\n  > ", num, c.Prompt)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, fmt.Errorf("unexpected end of input")
	}
	answer := strings.TrimSpace(scanner.Text())
	return strings.EqualFold(answer, strings.TrimSpace(c.Expected)), nil
}

func runKubectlOutput(num int, c *Check) (bool, error) {
	if c.Kubectl == "" {
		return false, fmt.Errorf("kubectl_output check %q missing kubectl field", c.ID)
	}
	fmt.Printf("  [%d] Running: kubectl %s\n", num, c.Kubectl)

	out, err := exec.Command("/bin/sh", "-c", "kubectl "+c.Kubectl).Output() //nolint:gosec
	if err != nil {
		return false, fmt.Errorf("kubectl: %w", err)
	}

	actual := strings.TrimSpace(string(out))

	if c.MatchType == "regex" {
		re, reErr := regexp.Compile(c.Expected)
		if reErr != nil {
			return false, fmt.Errorf("invalid regex in expected: %w", reErr)
		}
		return re.MatchString(actual), nil
	}

	return strings.EqualFold(actual, strings.TrimSpace(c.Expected)), nil
}

// ─── completion webhook ───────────────────────────────────────────────────────

type completionPayload struct {
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	LabID        string `json:"lab_id"`
	ChecksPassed int    `json:"checks_passed"`
	ChecksTotal  int    `json:"checks_total"`
}

func sendCompletion(webhookURL, secret, sessionID, userID, labID string, passed, total int) error {
	payload := completionPayload{
		SessionID:    sessionID,
		UserID:       userID,
		LabID:        labID,
		ChecksPassed: passed,
		ChecksTotal:  total,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gymctl-Signature", sig)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}

	return nil
}
