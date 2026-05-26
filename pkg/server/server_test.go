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

// --- /value -----------------------------------------------------------------

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

// --- /simulate --------------------------------------------------------------

func TestSimulateHandler_GBM(t *testing.T) {
	body := `{"model":"gbm","initial_rate":0.05,"drift":0.02,"volatility":0.15,"num_paths":100,"steps":10,"include_paths":true}`
	req := httptest.NewRequest("POST", "/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).simulateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp SimulateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Paths) != 100 {
		t.Errorf("len(paths) = %d, want 100", len(resp.Paths))
	}
}

func TestSimulateHandler_Vasicek(t *testing.T) {
	body := `{"model":"vasicek","initial_rate":0.05,"volatility":0.02,"long_term_mean":0.05,"mean_reversion":0.3,"num_paths":50,"steps":10,"include_paths":true}`
	req := httptest.NewRequest("POST", "/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).simulateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp SimulateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Paths) != 50 {
		t.Errorf("len(paths) = %d, want 50", len(resp.Paths))
	}
}

func TestSimulateHandler_Defaults(t *testing.T) {
	body := `{"initial_rate":0.05,"drift":0.02,"volatility":0.15}`
	req := httptest.NewRequest("POST", "/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).simulateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp SimulateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Paths != nil {
		t.Error("Paths should be nil when include_paths is false")
	}
	if resp.Mean == 0 {
		t.Error("Mean should be non-zero")
	}
}

func TestSimulateHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/simulate", nil)
	w := httptest.NewRecorder()
	New(":0").routes().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- /annuity ---------------------------------------------------------------

func TestAnnuityHandler_WholeLifeImmediate(t *testing.T) {
	// qx from age 0 through 110; young ages low, old ages high
	qxs := make([]float64, 111)
	for i := 0; i <= 110; i++ {
		switch {
		case i < 30:
			qxs[i] = 0.001
		case i < 50:
			qxs[i] = 0.003
		case i < 70:
			qxs[i] = 0.010
		case i < 90:
			qxs[i] = 0.050
		default:
			qxs[i] = 0.200
		}
	}
	qxsJSON, _ := json.Marshal(qxs)
	body := `{"interest_rate":0.05,"qxs":` + string(qxsJSON) + `,"age":30,"amount":1000,"computation":"whole_life_immediate"}`
	req := httptest.NewRequest("POST", "/annuity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).annuityHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp AnnuityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.PresentValue == 0 {
		t.Error("PresentValue should be non-zero")
	}
}

func TestAnnuityHandler_TermNSP(t *testing.T) {
	qxs := make([]float64, 111)
	for i := 0; i <= 110; i++ {
		switch {
		case i < 30:
			qxs[i] = 0.001
		case i < 50:
			qxs[i] = 0.003
		case i < 70:
			qxs[i] = 0.010
		case i < 90:
			qxs[i] = 0.050
		default:
			qxs[i] = 0.200
		}
	}
	qxsJSON, _ := json.Marshal(qxs)
	body := `{"interest_rate":0.05,"qxs":` + string(qxsJSON) + `,"age":30,"term":3,"amount":100000,"computation":"term_nsp"}`
	req := httptest.NewRequest("POST", "/annuity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).annuityHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp AnnuityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.PresentValue == 0 {
		t.Error("PresentValue should be non-zero")
	}
}

func TestAnnuityHandler_InvalidComputation(t *testing.T) {
	body := `{"interest_rate":0.05,"qxs":[0.001],"age":30,"amount":1000,"computation":"nonsense"}`
	req := httptest.NewRequest("POST", "/annuity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).annuityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAnnuityHandler_EmptyQxs(t *testing.T) {
	body := `{"interest_rate":0.05,"qxs":[],"age":30,"amount":1000,"computation":"term_immediate"}`
	req := httptest.NewRequest("POST", "/annuity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).annuityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- /reserve ---------------------------------------------------------------

func TestReserveHandler_NetPremium(t *testing.T) {
	qxs := make([]float64, 111)
	for i := 0; i <= 110; i++ {
		switch {
		case i < 30:
			qxs[i] = 0.001
		case i < 50:
			qxs[i] = 0.003
		case i < 70:
			qxs[i] = 0.010
		case i < 90:
			qxs[i] = 0.050
		default:
			qxs[i] = 0.200
		}
	}
	qxsJSON, _ := json.Marshal(qxs)
	body := `{"interest_rate":0.05,"qxs":` + string(qxsJSON) + `,"age":30,"term":3,"sum_assured":100000,"method":"net_premium"}`
	req := httptest.NewRequest("POST", "/reserve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).reserveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ReserveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Reserve == 0 {
		t.Error("Reserve should be non-zero")
	}
}

func TestReserveHandler_InvalidMethod(t *testing.T) {
	body := `{"interest_rate":0.05,"qxs":[0.001],"age":30,"term":3,"sum_assured":100000,"method":"nonsense"}`
	req := httptest.NewRequest("POST", "/reserve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).reserveHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReserveHandler_EmptyQxs(t *testing.T) {
	body := `{"interest_rate":0.05,"qxs":[],"age":30,"term":3,"sum_assured":100000,"method":"net_premium"}`
	req := httptest.NewRequest("POST", "/reserve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Server{}).reserveHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- /health wrong method test ----------------------------------------------

func TestHealthHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()
	New(":0").routes().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- misc -------------------------------------------------------------------

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
