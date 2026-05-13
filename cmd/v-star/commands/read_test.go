package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	exit = func(code int) {
		// Prevent os.Exit during tests
	}
	os.Exit(m.Run())
}

func writeTestCSV(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, line := range lines {
		f.WriteString(line)
		f.WriteString("\n")
	}
	f.Close()
	return path
}

func TestRead_CSVOutput(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,1000,1",
		"45,F,whole,2000,2",
	}
	path := writeTestCSV(t, lines)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Read([]string{"v-star", path, "--output=csv", "--limit=2"})

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	for {
		b := make([]byte, 4096)
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	output := buf.String()

	if !strings.Contains(output, "present_value") {
		t.Errorf("CSV header not found in output: %s", output)
	}
	if !strings.Contains(output, "1000.00") {
		t.Errorf("SumAssured 1000 not found in output: %s", output)
	}
}

func TestRead_InvalidInterestRate(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,1000,1",
	}
	path := writeTestCSV(t, lines)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Read([]string{"v-star", path, "--output=csv", "--limit=1", "--interest=abc"})

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	for {
		b := make([]byte, 4096)
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	output := buf.String()

	if !strings.Contains(output, "Warning: invalid interest rate") {
		t.Errorf("Warning message not found in output: %s", output)
	}
}

func TestRead_JSONOutput(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,1000,1",
	}
	path := writeTestCSV(t, lines)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Read([]string{"v-star", path, "--output=json", "--limit=1"})

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	for {
		b := make([]byte, 4096)
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	output := buf.String()

	if !strings.Contains(output, "sum_assured") {
		t.Errorf("JSON output not found: %s", output)
	}
}
