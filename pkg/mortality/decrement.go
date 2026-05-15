package mortality

// DecrementTable combines multiple causes of decrement (death, lapse, disability, etc.)
// into a single decrement table. Uses the multiple-decrement approach where
// each decrement has its own qx values and the total decrement probability
// at each age is 1 - product(1 - qx_i).
type DecrementTable struct {
	tables []*Table
	names  []string
	qx     []float64
	lx     []float64
	maxAge int
}

// NewDecrementTable creates a combined decrement table from multiple
// single-decrement tables. Each table represents one cause of decrement.
// Pre-computes combined qx and lx for O(1) lookups.
func NewDecrementTable(tables []*Table, names []string) *DecrementTable {
	if len(tables) == 0 {
		return &DecrementTable{maxAge: -1}
	}
	if len(names) == 0 {
		names = make([]string, len(tables))
		for i := range tables {
			names[i] = tables[i].Name()
		}
	}
	maxAge := tables[0].MaxAge()
	for _, t := range tables[1:] {
		if t.MaxAge() < maxAge {
			maxAge = t.MaxAge()
		}
	}
	qx := make([]float64, maxAge+1)
	for age := 0; age <= maxAge; age++ {
		survival := 1.0
		for _, t := range tables {
			survival *= 1 - t.Qx(age)
		}
		qx[age] = 1 - survival
	}
	lx := make([]float64, maxAge+2)
	lx[0] = 100000
	for i := 1; i < len(lx); i++ {
		lx[i] = lx[i-1] * (1 - qx[i-1])
	}
	return &DecrementTable{
		tables: tables,
		names:  names,
		qx:     qx,
		lx:     lx,
		maxAge: maxAge,
	}
}

// Qx returns the total probability of decrement at age: 1 - prod(1 - qx_i).
func (d *DecrementTable) Qx(age int) float64 {
	if d == nil || age < 0 || age > d.maxAge {
		return 0
	}
	return d.qx[age]
}

// QxByCause returns the probability of decrement from the specific cause at age.
// Since competing risks interact, this is approximate using the proportional
// approach: qx_cause * (1 - qx_total) / (1 - qx_cause).
func (d *DecrementTable) QxByCause(age int, causeIdx int) float64 {
	if d == nil || causeIdx < 0 || causeIdx >= len(d.tables) {
		return 0
	}
	causeQx := d.tables[causeIdx].Qx(age)
	totalQx := d.Qx(age)
	if totalQx <= 0 {
		return 0
	}
	return causeQx * (1 - totalQx) / (1 - causeQx)
}

// Px returns the total survival probability over term years: product(1 - qx_t) for t=age..age+term-1.
// Uses pre-computed lx table for O(1) lookup.
func (d *DecrementTable) Px(age int, term int) float64 {
	if d == nil || age < 0 || term <= 0 || age > d.maxAge || d.lx[age] == 0 {
		return 0
	}
	endAge := age + term
	if endAge >= len(d.lx) {
		return 0
	}
	return d.lx[endAge] / d.lx[age]
}

// MaxAge returns the minimum max age across all underlying tables.
func (d *DecrementTable) MaxAge() int {
	if d == nil {
		return -1
	}
	return d.maxAge
}

// Name returns the combined table name.
func (d *DecrementTable) Name() string {
	if d == nil {
		return ""
	}
	if len(d.names) == 0 {
		return "multiple-decrement"
	}
	return d.names[0]
}

// CauseNames returns the names of all causes of decrement.
func (d *DecrementTable) CauseNames() []string {
	return d.names
}

// NumCauses returns the number of decrement causes.
func (d *DecrementTable) NumCauses() int {
	return len(d.tables)
}
