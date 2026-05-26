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
	if resp.TotalPV == 0 {
		t.Error("TotalPV should be non-zero")
	}
}

func TestPVHandler_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/value", strings.NewReader(""))
	w := httptest.NewRecorder()
	(&Server{}).pvHandler(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected error for empty body")
	}
}

func TestPVHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/value", nil)
	w := httptest.NewRecorder()
	New(":0").routes().ServeHTTP(w, req)

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

func TestMonteCarloHandler_NoPaths(t *testing.T) {
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

func TestMonteCarloHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/montecarlo", nil)
	w := httptest.NewRecorder()
	New(":0").routes().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
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

func TestConvertRateHandler_Nominal(t *testing.T) {
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

func TestUploadCSVHandler(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "test.csv")
	part.Write([]byte("age,sex,policy_type,sum_assured,term\n30,M,term,1000,1\n"))
	mw.WriteField("rate", "0.05")
	mw.Close()

	req := httptest.NewRequest("POST", "/upload/csv", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	(&Server{}).uploadCSVHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "present_value") {
		t.Errorf("CSV output missing header: %s", w.Body.String())
	}
}

func TestUploadCSVHandler_NoFile(t *testing.T) {
	req := httptest.NewRequest("POST", "/upload/csv", nil)
	w := httptest.NewRecorder()
	(&Server{}).uploadCSVHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestServerNew(t *testing.T) {
	s := New(":8080")
	if s == nil {
		t.Error("New returned nil")
	}
}

func BenchmarkServerNew(b *testing.B) {
	for b.Loop() {
		_ = New(":8080")
	}
}
