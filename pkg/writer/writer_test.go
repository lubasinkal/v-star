package writer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lubasinkal/v-star/pkg/risk"
)

func TestCSVWriter_WriteHeader_Idempotent(t *testing.T) {
	var buf bytes.Buffer
	cw := NewCSVWriter(&buf)

	if err := cw.WriteHeader(); err != nil {
		t.Fatalf("first WriteHeader: %v", err)
	}
	first := buf.String()

	if err := cw.WriteHeader(); err != nil {
		t.Fatalf("second WriteHeader: %v", err)
	}
	second := buf.String()

	if first != second {
		t.Error("WriteHeader should be idempotent, output changed")
	}
}

func TestNewRecordWriter_CSV(t *testing.T) {
	var buf bytes.Buffer
	w := NewRecordWriter(&buf, "csv")

	if err := w.WriteRecord(Record{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20, PresentValue: 37688.95}); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "M,term,30,100000.00,20,37688.95") {
		t.Errorf("expected record in output, got: %s", output)
	}
}

func TestNewRecordWriter_JSON(t *testing.T) {
	var buf bytes.Buffer
	w := NewRecordWriter(&buf, "json")

	if err := w.WriteRecord(Record{Age: 30, Sex: "M", SumAssured: 100000, PresentValue: 37688.95}); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "[") || !strings.HasSuffix(output, "]") {
		t.Errorf("expected JSON array, got: %s", output)
	}
}

func TestNewRecordWriter_Default(t *testing.T) {
	var buf bytes.Buffer
	w := NewRecordWriter(&buf, "")

	if err := w.WriteRecord(Record{Age: 30, SumAssured: 100000, PresentValue: 37688.95}); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "[") {
		t.Errorf("expected JSON output for default format, got: %s", output)
	}
}

func TestStreamTextReport_DefaultTitle(t *testing.T) {
	var buf bytes.Buffer
	data := ReportData{InterestRate: 0.05, RecordCount: 10, TotalPresentValue: 100000}

	if err := StreamTextReport(data, &buf); err != nil {
		t.Fatalf("StreamTextReport: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Actuarial Valuation Report") {
		t.Errorf("expected default title, got: %s", output)
	}
}

func TestCSVRecord(t *testing.T) {
	record := CSVRecord{
		Sex:          "M",
		PolicyType:   "term",
		Age:          30,
		SumAssured:   100000,
		Term:         20,
		PresentValue: 37688.95,
	}

	var buf bytes.Buffer
	cw := NewCSVWriter(&buf)
	if err := cw.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if err := cw.WriteRecord(record); err != nil {
		t.Fatalf("WriteRecord failed: %v", err)
	}
	if err := cw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "sex,policy_type,age,sum_assured,term,present_value") {
		t.Errorf("CSV header not found in output: %s", output)
	}
	if !strings.Contains(output, "M") {
		t.Errorf("Sex 'M' not found in output: %s", output)
	}
	if !strings.Contains(output, "37688.95") {
		t.Errorf("PresentValue '37688.95' not found in output: %s", output)
	}
}

func TestStreamCSV(t *testing.T) {
	records := []CSVRecord{
		{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20, PresentValue: 37688.95},
		{Age: 45, Sex: "F", PolicyType: "whole", SumAssured: 200000, Term: 0, PresentValue: 78000.00},
	}

	var buf bytes.Buffer
	if err := StreamCSV(records, &buf); err != nil {
		t.Fatalf("StreamCSV failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines (header + 2 records), got %d", len(lines))
	}
}

func TestStreamTextReport(t *testing.T) {
	data := ReportData{
		Title:             "Test Valuation Report",
		InterestRate:      0.05,
		RecordCount:       100,
		TotalPresentValue: 5000000,
		Assumptions: map[string]string{
			"Interest Rate":   "5.00%",
			"Mortality Table": "CSO2017",
		},
	}

	var buf bytes.Buffer
	if err := StreamTextReport(data, &buf); err != nil {
		t.Fatalf("StreamTextReport failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test Valuation Report") {
		t.Errorf("Title not found in report: %s", output)
	}
	if !strings.Contains(output, "Records Processed: 100") {
		t.Errorf("Record count not found in report: %s", output)
	}
	if !strings.Contains(output, "Total Present Value: 5000000.00") {
		t.Errorf("Total PV not found in report: %s", output)
	}
}

func TestStreamTextReportWithRisk(t *testing.T) {
	data := ReportData{
		Title:             "Risk Analysis Report",
		InterestRate:      0.05,
		RecordCount:       1000,
		TotalPresentValue: 50000000,
		RiskReport: &risk.RiskReport{
			Mean:   100000,
			StdDev: 15000,
			Min:    70000,
			Max:    130000,
			VaR95:  125000,
			VaR99:  129000,
			CTE95:  126000,
			CTE99:  130000,
		},
	}

	var buf bytes.Buffer
	if err := StreamTextReport(data, &buf); err != nil {
		t.Fatalf("StreamTextReport failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Risk Analysis") {
		t.Errorf("Risk Analysis section not found: %s", output)
	}
	if !strings.Contains(output, "VaR 95%") {
		t.Errorf("VaR 95%% not found in report: %s", output)
	}
	if !strings.Contains(output, "CTE 99%") {
		t.Errorf("CTE 99%% not found in report: %s", output)
	}
}

func TestFormatAssumptions(t *testing.T) {
	assumptions := FormatAssumptions(0.05, "CSO2017", map[string]string{
		"Expenses": "2.5%",
	})

	if assumptions["Interest Rate"] != "5.00%" {
		t.Errorf("Expected Interest Rate 5.00%%, got %s", assumptions["Interest Rate"])
	}
	if assumptions["Mortality Table"] != "CSO2017" {
		t.Errorf("Expected Mortality Table CSO2017, got %s", assumptions["Mortality Table"])
	}
	if assumptions["Expenses"] != "2.5%" {
		t.Errorf("Expected Expenses 2.5%%, got %s", assumptions["Expenses"])
	}
}

func TestGenerateTextReport(t *testing.T) {
	data := ReportData{
		Title:             "Generated Report Test",
		InterestRate:      0.04,
		RecordCount:       50,
		TotalPresentValue: 2500000,
		Assumptions: map[string]string{
			"Interest Rate":   "4.00%",
			"Mortality Table": "IAM2012",
		},
	}
	report, err := GenerateTextReport(data)
	if err != nil {
		t.Fatalf("GenerateTextReport failed: %v", err)
	}
	if !strings.Contains(report, "Generated Report Test") {
		t.Errorf("Title not found")
	}
	if !strings.Contains(report, "Records Processed: 50") {
		t.Errorf("Record count not found")
	}
}

func TestGenerateTextReportWithRisk(t *testing.T) {
	data := ReportData{
		Title:             "Risk Report Generated",
		InterestRate:      0.05,
		RecordCount:       500,
		TotalPresentValue: 10000000,
		RiskReport: &risk.RiskReport{
			Mean: 50000, StdDev: 8000, Min: 35000, Max: 65000,
			VaR95: 62000, VaR99: 64000, CTE95: 63000, CTE99: 64500,
		},
	}
	report, err := GenerateTextReport(data)
	if err != nil {
		t.Fatalf("GenerateTextReport failed: %v", err)
	}
	if !strings.Contains(report, "Risk Report Generated") {
		t.Errorf("Title not found")
	}
	if !strings.Contains(report, "VaR 95%") {
		t.Errorf("VaR 95%% not found")
	}
}

func TestGenerateTextReportEmpty(t *testing.T) {
	report, err := GenerateTextReport(ReportData{})
	if err != nil {
		t.Fatalf("GenerateTextReport failed: %v", err)
	}
	if !strings.Contains(report, "Actuarial Valuation Report") {
		t.Errorf("Default title not found")
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text\nwith\nnewlines", "text with newlines"},
		{"text\rwith\rcarriage", "text with carriage"},
		{"text\twith\ttabs", "text with tabs"},
	}

	for _, tt := range tests {
		result := SanitizeString(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestJSONRecord(t *testing.T) {
	record := JSONRecord{
		Sex:          "M",
		PolicyType:   "term",
		Age:          30,
		SumAssured:   100000,
		Term:         20,
		PresentValue: 37688.95,
	}

	var buf bytes.Buffer
	jw := NewJSONWriter(&buf)
	if err := jw.WriteRecord(record); err != nil {
		t.Fatalf("WriteRecord failed: %v", err)
	}
	if err := jw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"sex":"M"`) {
		t.Errorf("Sex not found in JSON output: %s", output)
	}
	if !strings.Contains(output, `"present_value":37688.95`) {
		t.Errorf("PresentValue not found in JSON output: %s", output)
	}
}
