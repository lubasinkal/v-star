package mortality

import (
	"math"
	"testing"
)

func TestDecrementTable_Qx(t *testing.T) {
	deathQx := []float64{0.01, 0.02, 0.03}
	lapseQx := []float64{0.05, 0.04, 0.03}

	t1 := NewTable("death", deathQx)
	t2 := NewTable("lapse", lapseQx)
	dt := NewDecrementTable([]*Table{t1, t2}, []string{"death", "lapse"})

	// At age 0: total qx = 1 - (1-0.01)*(1-0.05) = 1 - 0.99*0.95 = 1 - 0.9405 = 0.0595
	expected := 1 - (1-0.01)*(1-0.05)
	got := dt.Qx(0)
	if math.Abs(got-expected) > 1e-10 {
		t.Errorf("Qx(0) = %.10f, want %.10f", got, expected)
	}
}

func TestDecrementTable_Px(t *testing.T) {
	t1 := NewTable("death", []float64{0.01, 0.01})
	t2 := NewTable("lapse", []float64{0.05, 0.05})
	dt := NewDecrementTable([]*Table{t1, t2}, nil)

	// Px(0, 2) = (1-qx[0]) * (1-qx[1])
	// qx[0] = 1 - (1-0.01)*(1-0.05) = 0.0595
	// qx[1] = same = 0.0595
	// Px = (1-0.0595)^2 = 0.8845...
	survival := 1 - dt.Qx(0)
	expected := survival * survival
	got := dt.Px(0, 2)
	if math.Abs(got-expected) > 1e-10 {
		t.Errorf("Px(0,2) = %.10f, want %.10f", got, expected)
	}
}

func TestDecrementTable_MaxAge(t *testing.T) {
	t1 := NewTable("death", []float64{0.01, 0.02, 0.03, 0.04})
	t2 := NewTable("lapse", []float64{0.05, 0.04, 0.03, 0.02})
	dt := NewDecrementTable([]*Table{t1, t2}, nil)
	if dt.MaxAge() != 3 {
		t.Errorf("MaxAge() = %d, want 3", dt.MaxAge())
	}
}

func TestDecrementTable_SingleTable(t *testing.T) {
	t1 := NewTable("death", []float64{0.01, 0.02})
	dt := NewDecrementTable([]*Table{t1}, nil)
	got := dt.Qx(0)
	if math.Abs(got-0.01) > 1e-10 {
		t.Errorf("Single table Qx = %.15f, want 0.01", got)
	}
}

func TestDecrementTable_Name(t *testing.T) {
	t1 := NewTable("death", []float64{0.01})
	dt := NewDecrementTable([]*Table{t1}, []string{"death"})
	if dt.Name() != "death" {
		t.Errorf("Name() = %q, want %q", dt.Name(), "death")
	}
}

func TestDecrementTable_Nil(t *testing.T) {
	var dt *DecrementTable
	if dt.Qx(0) != 0 {
		t.Error("nil DecrementTable should return 0 for Qx")
	}
	if dt.MaxAge() != -1 {
		t.Error("nil DecrementTable should return -1 for MaxAge")
	}
}

func TestDecrementTable_Empty(t *testing.T) {
	dt := NewDecrementTable(nil, nil)
	if dt.Qx(0) != 0 {
		t.Error("empty DecrementTable should return 0 for Qx")
	}
}

func TestDecrementTable_QxByCause(t *testing.T) {
	t1 := NewTable("death", []float64{0.01, 0.02})
	t2 := NewTable("lapse", []float64{0.05, 0.04})
	dt := NewDecrementTable([]*Table{t1, t2}, []string{"death", "lapse"})

	causeQx := dt.QxByCause(0, 0)
	if causeQx <= 0 {
		t.Errorf("QxByCause(0, death) = %v, want > 0", causeQx)
	}
	causeQx = dt.QxByCause(0, 5)
	if causeQx != 0 {
		t.Errorf("QxByCause(0, invalid) = %v, want 0", causeQx)
	}
}
