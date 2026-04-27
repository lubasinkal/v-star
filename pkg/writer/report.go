package writer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/lubasinkal/v-star/pkg/risk"
)

// ReportData contains all data needed for an actuarial report
type ReportData struct {
	Title             string
	GeneratedAt       string
	InterestRate      float64
	RecordCount       int
	TotalPresentValue float64
	RiskReport        *risk.RiskReport
	Assumptions       map[string]string
}

// DefaultTextReportTemplate is the default template for text reports
const DefaultTextReportTemplate = `{{.Title}}
================================

Generated: {{.GeneratedAt}}
Interest Rate: {{printf "%.2f" (mul .InterestRate 100)}}%

Summary
--------
Records Processed: {{.RecordCount}}
Total Present Value: {{printf "%.2f" .TotalPresentValue}}

{{if .RiskReport}}
Risk Analysis
--------------
Mean: {{printf "%.4f" .RiskReport.Mean}}
Std Dev: {{printf "%.4f" .RiskReport.StdDev}}
Min: {{printf "%.4f" .RiskReport.Min}}
Max: {{printf "%.4f" .RiskReport.Max}}
VaR 95%: {{printf "%.4f" .RiskReport.VaR95}}
VaR 99%: {{printf "%.4f" .RiskReport.VaR99}}
CTE 95%: {{printf "%.4f" .RiskReport.CTE95}}
CTE 99%: {{printf "%.4f" .RiskReport.CTE99}}
{{end}}
{{if .Assumptions}}
Assumptions
-----------
{{range $key, $value := .Assumptions}}
{{$key}}: {{$value}}
{{end}}
{{end}}
`

// StreamTextReport writes a text report to w using the default template
func StreamTextReport(data ReportData, w io.Writer) error {
	if data.GeneratedAt == "" {
		data.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	if data.Title == "" {
		data.Title = "Actuarial Valuation Report"
	}

	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(DefaultTextReportTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

// GenerateTextReport returns the text report as a string
func GenerateTextReport(data ReportData) (string, error) {
	var buf bytes.Buffer
	if err := StreamTextReport(data, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FormatAssumptions creates a formatted assumptions map
func FormatAssumptions(interestRate float64, mortalityTable string, additional map[string]string) map[string]string {
	assumptions := map[string]string{
		"Interest Rate": fmt.Sprintf("%.2f%%", interestRate*100),
	}
	if mortalityTable != "" {
		assumptions["Mortality Table"] = mortalityTable
	}
	for k, v := range additional {
		assumptions[k] = v
	}
	return assumptions
}

// SanitizeString removes newlines and other problematic characters for CSV/Reports
func SanitizeString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}
