package server

import (
	"encoding/json"
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
	if len(resp.Paths) != 100 {
		t.Errorf("len(paths) = %d, want 100", len(resp.Paths))
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

func BenchmarkServerNew(b *testing.B) {
	for b.Loop() {
		_ = New(":8080")
	}
}
