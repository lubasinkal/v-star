package mortality

import (
	"math"
	"testing"
)

func TestTableQx(t *testing.T) {
	qx := []float64{0.001, 0.002, 0.005, 0.01, 0.02}
	table := NewTable("test", qx)

	tests := []struct {
		age  int
		want float64
	}{
		{0, 0.001},
		{1, 0.002},
		{2, 0.005},
		{3, 0.01},
		{4, 0.02},
		{-1, 0},
		{5, 0},
		{100, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := table.Qx(tt.age)
			if got != tt.want {
				t.Errorf("Qx(%d) = %v, want %v", tt.age, got, tt.want)
			}
		})
	}
}

func TestTablePx(t *testing.T) {
	qx := []float64{0.0, 0.0, 0.0, 0.0, 0.0}
	table := NewTable("test", qx)

	got := table.Px(0, 3)
	want := 1.0
	if got != want {
		t.Errorf("Px(0, 3) = %v, want %v", got, want)
	}

	qx2 := []float64{0.1, 0.1, 0.1}
	table2 := NewTable("test2", qx2)

	got2 := table2.Px(0, 2)
	want2 := 0.9 * 0.9
	if math.Abs(got2-want2) > 1e-9 {
		t.Errorf("Px(0, 2) = %v, want %v", got2, want2)
	}

	got3 := table2.Px(0, 0)
	if got3 != 1.0 {
		t.Errorf("Px(0, 0) = %v, want 1.0", got3)
	}

	got4 := table2.Px(0, -1)
	if got4 != 1.0 {
		t.Errorf("Px(0, -1) = %v, want 1.0", got4)
	}
}

func TestTableLx(t *testing.T) {
	qx := []float64{0.01, 0.01, 0.01}
	table := NewTable("test", qx)

	if table.Lx(0) != 100000 {
		t.Errorf("Lx(0) = %v, want 100000", table.Lx(0))
	}

	got := table.Lx(1)
	want := 100000 * 0.99
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Lx(1) = %v, want %v", got, want)
	}
}

func TestTableMaxAge(t *testing.T) {
	qx := []float64{0.001, 0.002, 0.005}
	table := NewTable("test", qx)

	if table.MaxAge() != 2 {
		t.Errorf("MaxAge() = %v, want 2", table.MaxAge())
	}
}

func TestTableName(t *testing.T) {
	qx := []float64{0.001}
	table := NewTable("CD2021", qx)

	if table.Name() != "CD2021" {
		t.Errorf("Name() = %v, want CD2021", table.Name())
	}
}

func TestTableEx(t *testing.T) {
	qx := []float64{0.0, 0.0, 0.0, 0.0, 0.0}
	table := NewTable("test", qx)

	got := table.Ex(0)
	want := 4.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Ex(0) = %v, want %v", got, want)
	}

	got2 := table.Ex(-1)
	if got2 != 0 {
		t.Errorf("Ex(-1) = %v, want 0", got2)
	}

	got3 := table.Ex(4)
	if got3 != 0 {
		t.Errorf("Ex(4) = %v, want 0 (cannot survive beyond max age)", got3)
	}
}

func BenchmarkQx(b *testing.B) {
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = float64(i) * 0.0001
	}
	table := NewTable("bench", qx)

	for b.Loop() {
		for age := 0; age <= 120; age++ {
			_ = table.Qx(age)
		}
	}
}

func BenchmarkPx(b *testing.B) {
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = float64(i) * 0.0001
	}
	table := NewTable("bench", qx)

	for b.Loop() {
		for age := 0; age <= 100; age++ {
			_ = table.Px(age, 20)
		}
	}
}
