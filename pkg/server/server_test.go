package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	(&Server{}).healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestPVHandlerInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/value", nil)
	w := httptest.NewRecorder()

	(&Server{}).pvHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPVHandlerEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/value", strings.NewReader(""))
	w := httptest.NewRecorder()

	(&Server{}).pvHandler(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected error for empty body")
	}
}

func TestPVHandler(t *testing.T) {
	body := `{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}`
	req := httptest.NewRequest("POST", "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).pvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp PVResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.RecordCount != 1 {
		t.Errorf("RecordCount = %d, want 1", resp.RecordCount)
	}
}

func TestMonteCarloHandlerInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/montecarlo", nil)
	w := httptest.NewRecorder()

	(&Server{}).monteCarloHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMonteCarloHandler(t *testing.T) {
	body := `{"initial_rate":0.05,"drift":0.02,"volatility":0.15,"num_paths":100,"steps":10,"include_paths":true}`
	req := httptest.NewRequest("POST", "/montecarlo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).monteCarloHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MonteCarloResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Paths) != 100 {
		t.Errorf("len(paths) = %d, want 100", len(resp.Paths))
	}
}

func TestMonteCarloHandlerNoPaths(t *testing.T) {
	body := `{"initial_rate":0.05,"drift":0.02,"volatility":0.15,"num_paths":100,"steps":10}`
	req := httptest.NewRequest("POST", "/montecarlo", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).monteCarloHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp MonteCarloResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Paths != nil {
		t.Errorf("Paths should be nil when include_paths is false")
	}
	if resp.Mean == 0 {
		t.Error("Mean should be non-zero")
	}
}

func TestConvertRateHandler(t *testing.T) {
	body := `{"from_rate":0.05,"from_type":"effective","compounding":1}`
	req := httptest.NewRequest("POST", "/convert-rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).convertRateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ConvertRateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.EffectiveRate != 0.05 {
		t.Errorf("EffectiveRate = %v, want 0.05", resp.EffectiveRate)
	}
}

func TestConvertRateHandlerNominal(t *testing.T) {
	body := `{"from_rate":0.049,"from_type":"nominal","compounding":1}`
	req := httptest.NewRequest("POST", "/convert-rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).convertRateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ConvertRateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.NominalRate != 0.049 {
		t.Errorf("NominalRate = %v, want 0.049", resp.NominalRate)
	}
}

func TestMortalityHandlerEmptyTable(t *testing.T) {
	req := httptest.NewRequest("GET", "/mortality/", nil)
	w := httptest.NewRecorder()

	(&Server{}).mortalityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMortalityHandlerNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/mortality/nonexistent", nil)
	w := httptest.NewRecorder()

	(&Server{}).mortalityHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServerNew(t *testing.T) {
	s := New(":8080")
	if s == nil {
		t.Error("New returned nil")
	}
}

func TestExportCSVHandlerInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/export/csv", nil)
	w := httptest.NewRecorder()

	(&Server{}).exportCSVHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestExportCSVHandler(t *testing.T) {
	body := `{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}`
	req := httptest.NewRequest("POST", "/export/csv", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).exportCSVHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/csv")
	}

	if !strings.Contains(w.Body.String(), "sex,policy_type,age,sum_assured,term,present_value") {
		t.Errorf("CSV header not found in response: %s", w.Body.String())
	}
}

func TestExportReportHandlerInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/export/report", nil)
	w := httptest.NewRecorder()

	(&Server{}).exportReportHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestStreamCSVHandler(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.csv")
	part.Write([]byte("age,sex,policy_type,sum_assured,term\n30,M,term,1000,1\n"))
	writer.WriteField("rate", "0.05")
	writer.Close()

	req := httptest.NewRequest("POST", "/upload/csv", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	s := &Server{MortalityTableDir: "testdata"}
	s.StreamCSVHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "present_value") {
		t.Errorf("CSV output missing header: %s", w.Body.String())
	}
}

func TestStreamCSVHandler_NoFile(t *testing.T) {
	req := httptest.NewRequest("POST", "/upload/csv", nil)
	w := httptest.NewRecorder()
	s := &Server{}
	s.StreamCSVHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMortalityHandler_WithTable(t *testing.T) {
	// No mortality table file to test against, so expect not-found
	req := httptest.NewRequest("GET", "/mortality/testtable", nil)
	w := httptest.NewRecorder()
	s := &Server{MortalityTableDir: "/nonexistent"}
	s.mortalityHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExportReportHandler(t *testing.T) {
	body := `{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}`
	req := httptest.NewRequest("POST", "/export/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	(&Server{}).exportReportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/plain")
	}

	if !strings.Contains(w.Body.String(), "Actuarial Valuation Report") {
		t.Errorf("Report title not found in response: %s", w.Body.String())
	}
}

func BenchmarkServerNew(b *testing.B) {
	for b.Loop() {
		_ = New(":8080")
	}
}
