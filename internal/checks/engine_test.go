package checks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gymctl/internal/scenario"
)

func TestCompareValue(t *testing.T) {
	tests := []struct {
		name      string
		actual    string
		operator  string
		expected  interface{}
		valueType string
		wantPass  bool
	}{
		// equals operator
		{"equals match", "hello", "equals", "hello", "", true},
		{"equals mismatch", "hello", "equals", "world", "", false},
		{"equals empty strings", "", "equals", "", "", true},
		{"equals default operator", "test", "", "test", "", true},

		// notEquals operator
		{"notEquals match", "hello", "notEquals", "world", "", true},
		{"notEquals mismatch", "hello", "notEquals", "hello", "", false},

		// contains operator
		{"contains match", "hello world", "contains", "world", "", true},
		{"contains mismatch", "hello world", "contains", "foo", "", false},
		{"contains empty", "hello", "contains", "", "", true},

		// regex operator
		{"regex match simple", "hello123", "regex", `hello\d+`, "", true},
		{"regex mismatch", "hello", "regex", `\d+`, "", false},
		{"regex match complex", "error: file not found", "regex", `error:.*not found`, "", true},

		// exists operator
		{"exists with value", "something", "exists", nil, "", true},
		{"exists empty", "", "exists", nil, "", false},

		// greaterThan with number
		{"greaterThan number pass", "10", "greaterThan", "5", "number", true},
		{"greaterThan number fail", "3", "greaterThan", "5", "number", false},
		{"greaterThan number equal", "5", "greaterThan", "5", "number", false},

		// lessThan with number
		{"lessThan number pass", "3", "lessThan", "5", "number", true},
		{"lessThan number fail", "10", "lessThan", "5", "number", false},

		// greaterThan with quantity (kubernetes resource quantities)
		{"greaterThan quantity pass", "200Mi", "greaterThan", "100Mi", "quantity", true},
		{"greaterThan quantity fail", "50Mi", "greaterThan", "100Mi", "quantity", false},
		{"lessThan quantity pass", "50Mi", "lessThan", "100Mi", "quantity", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := compareValue(tt.actual, tt.operator, tt.expected, tt.valueType)
			if passed != tt.wantPass {
				t.Errorf("compareValue() passed = %v, want %v, msg = %s", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestCompareInt(t *testing.T) {
	tests := []struct {
		name     string
		actual   int64
		expected int64
		operator string
		wantPass bool
	}{
		{"equals match", 10, 10, "equals", true},
		{"equals mismatch", 10, 20, "equals", false},
		{"equals default", 5, 5, "", true},
		{"notEquals match", 10, 20, "notEquals", true},
		{"notEquals mismatch", 10, 10, "notEquals", false},
		{"greaterThan pass", 20, 10, "greaterThan", true},
		{"greaterThan fail", 5, 10, "greaterThan", false},
		{"greaterThan equal", 10, 10, "greaterThan", false},
		{"lessThan pass", 5, 10, "lessThan", true},
		{"lessThan fail", 20, 10, "lessThan", false},
		{"lessThan equal", 10, 10, "lessThan", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := compareInt(tt.actual, tt.expected, tt.operator)
			if passed != tt.wantPass {
				t.Errorf("compareInt() passed = %v, want %v, msg = %s", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{"bytes", "100B", 100, false},
		{"bytes lowercase", "100b", 100, false},
		{"kilobytes", "1KB", 1024, false},
		{"megabytes", "1MB", 1024 * 1024, false},
		{"gigabytes", "1GB", 1024 * 1024 * 1024, false},
		{"fractional MB", "1.5MB", int64(1.5 * 1024 * 1024), false},
		{"no suffix", "1024", 1024, false},
		{"with spaces", " 100MB ", 100 * 1024 * 1024, false},
		{"empty", "", 0, true},
		{"invalid", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSize(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckExpectOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expect   *scenario.ExpectOutput
		wantPass bool
	}{
		{
			name:     "contains match",
			output:   "hello world",
			expect:   &scenario.ExpectOutput{Contains: "world"},
			wantPass: true,
		},
		{
			name:     "contains mismatch",
			output:   "hello world",
			expect:   &scenario.ExpectOutput{Contains: "foo"},
			wantPass: false,
		},
		{
			name:     "notContains match",
			output:   "hello world",
			expect:   &scenario.ExpectOutput{NotContains: "foo"},
			wantPass: true,
		},
		{
			name:     "notContains mismatch",
			output:   "hello world",
			expect:   &scenario.ExpectOutput{NotContains: "world"},
			wantPass: false,
		},
		{
			name:     "regex match",
			output:   "error code: 123",
			expect:   &scenario.ExpectOutput{Regex: `code: \d+`},
			wantPass: true,
		},
		{
			name:     "regex mismatch",
			output:   "error code: abc",
			expect:   &scenario.ExpectOutput{Regex: `code: \d+`},
			wantPass: false,
		},
		{
			name:     "multiple conditions all pass",
			output:   "success: operation completed",
			expect:   &scenario.ExpectOutput{Contains: "success", NotContains: "error"},
			wantPass: true,
		},
		{
			name:     "multiple conditions one fails",
			output:   "error: operation failed",
			expect:   &scenario.ExpectOutput{Contains: "operation", NotContains: "error"},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExpectOutput(tt.output, tt.expect, Result{Name: "test"})
			if result.Passed != tt.wantPass {
				t.Errorf("checkExpectOutput() passed = %v, want %v, msg = %s", result.Passed, tt.wantPass, result.Message)
			}
		})
	}
}

func TestRunFileCheck(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "gymctl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\nline two"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name     string
		check    scenario.Check
		wantPass bool
	}{
		{
			name:     "file exists",
			check:    scenario.Check{Name: "test", Path: "test.txt", Exists: boolPtr(true)},
			wantPass: true,
		},
		{
			name:     "file does not exist",
			check:    scenario.Check{Name: "test", Path: "nonexistent.txt", Exists: boolPtr(true)},
			wantPass: false,
		},
		{
			name:     "file should not exist and doesn't",
			check:    scenario.Check{Name: "test", Path: "nonexistent.txt", Exists: boolPtr(false)},
			wantPass: true,
		},
		{
			name:     "file should not exist but does",
			check:    scenario.Check{Name: "test", Path: "test.txt", Exists: boolPtr(false)},
			wantPass: false,
		},
		{
			name:     "file content contains",
			check:    scenario.Check{Name: "test", Path: "test.txt", Operator: "contains", Value: "hello"},
			wantPass: true,
		},
		{
			name:     "file content does not contain",
			check:    scenario.Check{Name: "test", Path: "test.txt", Operator: "contains", Value: "goodbye"},
			wantPass: false,
		},
		{
			name:     "file content regex",
			check:    scenario.Check{Name: "test", Path: "test.txt", Operator: "regex", Value: `line \w+`},
			wantPass: true,
		},
		{
			name:     "directory exists",
			check:    scenario.Check{Name: "test", Path: "testdir", Exists: boolPtr(true)},
			wantPass: true,
		},
		{
			name:     "missing path",
			check:    scenario.Check{Name: "test", Path: ""},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runFileCheck(tt.check, tmpDir)
			if result.Passed != tt.wantPass {
				t.Errorf("runFileCheck() passed = %v, want %v, msg = %s", result.Passed, tt.wantPass, result.Message)
			}
		})
	}
}

func TestRunDockerfileCheck(t *testing.T) {
	// Create a temp directory for test Dockerfiles
	tmpDir, err := os.MkdirTemp("", "gymctl-dockerfile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simple single-stage Dockerfile
	singleStage := `FROM alpine:3.18
RUN apk add --no-cache curl
COPY . /app
CMD ["./app"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile.single"), []byte(singleStage), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	// Multi-stage Dockerfile
	multiStage := `FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o app

FROM alpine:3.18
COPY --from=builder /app/app /app
USER nobody
CMD ["/app"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile.multi"), []byte(multiStage), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	// Dockerfile without USER instruction
	noUser := `FROM alpine:3.18
RUN apk add --no-cache curl
CMD ["sh"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile.nouser"), []byte(noUser), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	tests := []struct {
		name     string
		check    scenario.Check
		wantPass bool
	}{
		{
			name:     "single stage is not multi-stage",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.single", Check: "multiStage", Operator: "equals", Value: "false"},
			wantPass: true,
		},
		{
			name:     "multi stage is multi-stage",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.multi", Check: "multiStage", Operator: "equals", Value: "true"},
			wantPass: true,
		},
		{
			name:     "base image check",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.single", Check: "baseImage", Operator: "contains", Value: "alpine"},
			wantPass: true,
		},
		{
			name:     "base image with tag",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.single", Check: "baseImage", Operator: "equals", Value: "alpine:3.18"},
			wantPass: true,
		},
		{
			name:     "user instruction exists",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.multi", Check: "userInstruction", Operator: "exists"},
			wantPass: true,
		},
		{
			name:     "user instruction missing",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.nouser", Check: "userInstruction", Operator: "exists"},
			wantPass: false,
		},
		{
			name:     "copy from exists in multi-stage",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.multi", Check: "copyFrom", Operator: "equals", Value: "true"},
			wantPass: true,
		},
		{
			name:     "copy from missing in single-stage",
			check:    scenario.Check{Name: "test", Path: "Dockerfile.single", Check: "copyFrom", Operator: "equals", Value: "false"},
			wantPass: true,
		},
		{
			name:     "missing dockerfile path",
			check:    scenario.Check{Name: "test", Path: ""},
			wantPass: false,
		},
		{
			name:     "nonexistent dockerfile",
			check:    scenario.Check{Name: "test", Path: "nonexistent", Check: "multiStage"},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runDockerfileCheck(tt.check, tmpDir)
			if result.Passed != tt.wantPass {
				t.Errorf("runDockerfileCheck() passed = %v, want %v, msg = %s", result.Passed, tt.wantPass, result.Message)
			}
		})
	}
}

func TestRunHTTPCheck(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal error"}`))
		case "/headers":
			if r.Header.Get("X-Custom") == "test" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("missing header"))
			}
		case "/post":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("created"))
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	status200 := 200
	status201 := 201
	status500 := 500

	tests := []struct {
		name     string
		check    scenario.Check
		wantPass bool
	}{
		{
			name:     "health check success",
			check:    scenario.Check{Name: "test", URL: server.URL + "/health", ExpectStatus: &status200},
			wantPass: true,
		},
		{
			name:     "health check body contains",
			check:    scenario.Check{Name: "test", URL: server.URL + "/health", ExpectBody: &scenario.ExpectBody{Contains: "healthy"}},
			wantPass: true,
		},
		{
			name:     "health check body regex",
			check:    scenario.Check{Name: "test", URL: server.URL + "/health", ExpectBody: &scenario.ExpectBody{Regex: `"status":\s*"healthy"`}},
			wantPass: true,
		},
		{
			name:     "error endpoint status",
			check:    scenario.Check{Name: "test", URL: server.URL + "/error", ExpectStatus: &status500},
			wantPass: true,
		},
		{
			name:     "wrong status code",
			check:    scenario.Check{Name: "test", URL: server.URL + "/health", ExpectStatus: &status500},
			wantPass: false,
		},
		{
			name:     "body not contains",
			check:    scenario.Check{Name: "test", URL: server.URL + "/health", ExpectBody: &scenario.ExpectBody{NotContains: "error"}},
			wantPass: true,
		},
		{
			name:     "body should not contain but does",
			check:    scenario.Check{Name: "test", URL: server.URL + "/error", ExpectBody: &scenario.ExpectBody{NotContains: "error"}},
			wantPass: false,
		},
		{
			name:     "custom headers",
			check:    scenario.Check{Name: "test", URL: server.URL + "/headers", Headers: map[string]string{"X-Custom": "test"}, ExpectStatus: &status200},
			wantPass: true,
		},
		{
			name:     "missing custom header",
			check:    scenario.Check{Name: "test", URL: server.URL + "/headers", ExpectStatus: &status200},
			wantPass: false,
		},
		{
			name:     "POST method",
			check:    scenario.Check{Name: "test", URL: server.URL + "/post", Method: "POST", ExpectStatus: &status201},
			wantPass: true,
		},
		{
			name:     "missing URL",
			check:    scenario.Check{Name: "test", URL: ""},
			wantPass: false,
		},
		{
			name:     "invalid URL",
			check:    scenario.Check{Name: "test", URL: "not-a-url"},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runHTTPCheck(ctx, tt.check)
			if result.Passed != tt.wantPass {
				t.Errorf("runHTTPCheck() passed = %v, want %v, msg = %s", result.Passed, tt.wantPass, result.Message)
			}
		})
	}
}

func TestRunScriptCheck(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "gymctl-script-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exitCode0 := 0
	exitCode1 := 1

	tests := []struct {
		name     string
		check    scenario.Check
		wantPass bool
	}{
		{
			name:     "simple echo",
			check:    scenario.Check{Name: "test", Script: "echo hello"},
			wantPass: true,
		},
		{
			name:     "check output contains",
			check:    scenario.Check{Name: "test", Script: "echo 'hello world'", ExpectOutput: &scenario.ExpectOutput{Contains: "world"}},
			wantPass: true,
		},
		{
			name:     "check output not contains",
			check:    scenario.Check{Name: "test", Script: "echo 'hello world'", ExpectOutput: &scenario.ExpectOutput{NotContains: "foo"}},
			wantPass: true,
		},
		{
			name:     "exit code 0",
			check:    scenario.Check{Name: "test", Script: "exit 0", ExpectExitCode: &exitCode0},
			wantPass: true,
		},
		{
			name:     "exit code 1 expected",
			check:    scenario.Check{Name: "test", Script: "exit 1", ExpectExitCode: &exitCode1},
			wantPass: true,
		},
		{
			name:     "exit code mismatch",
			check:    scenario.Check{Name: "test", Script: "exit 1", ExpectExitCode: &exitCode0},
			wantPass: false,
		},
		{
			name:     "script failure without expectExitCode",
			check:    scenario.Check{Name: "test", Script: "exit 1"},
			wantPass: false,
		},
		{
			name:     "missing script",
			check:    scenario.Check{Name: "test", Script: ""},
			wantPass: false,
		},
		{
			name:     "command with pipe",
			check:    scenario.Check{Name: "test", Script: "echo -e 'line1\nline2\nline3' | wc -l", ExpectOutput: &scenario.ExpectOutput{Contains: "3"}},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runScriptCheck(ctx, tt.check, tmpDir)
			if result.Passed != tt.wantPass {
				t.Errorf("runScriptCheck() passed = %v, want %v, msg = %s", result.Passed, tt.wantPass, result.Message)
			}
		})
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
