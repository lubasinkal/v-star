package mortality

// DecrementTable combines multiple causes of decrement (death, lapse, disability, etc.)
// into a single decrement table. Uses the multiple-decrement approach where
// each decrement has its own qx values and the total decrement probability
// at each age is 1 - product(1 - qx_i).
type DecrementTable struct {
	tables []*Table
	names  []string
}

// NewDecrementTable creates a combined decrement table from multiple
// single-decrement tables. Each table represents one cause of decrement.
// The resulting qx at each age = 1 - product(1 - qx_i) for all causes i.
func NewDecrementTable(tables []*Table, names []string) *DecrementTable {
	if len(tables) == 0 {
		return &DecrementTable{}
	}
	if len(names) == 0 {
		names = make([]string, len(tables))
		for i := range tables {
			names[i] = tables[i].Name()
		}
	}
	return &DecrementTable{tables: tables, names: names}
}

// Qx returns the total probability of decrement at age: 1 - prod(1 - qx_i).
func (d *DecrementTable) Qx(age int) float64 {
	if d == nil || len(d.tables) == 0 {
		return 0
	}
	survival := 1.0
	for _, t := range d.tables {
		survival *= 1 - t.Qx(age)
	}
	return 1 - survival
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

// Px returns the total survival probability: product(1 - qx_i) for all causes.
func (d *DecrementTable) Px(age int, term int) float64 {
	if d == nil || len(d.tables) == 0 {
		return 0
	}
	product := 1.0
	for t := 0; t < term; t++ {
		product *= 1 - d.Qx(age+t)
	}
	return product
}

// MaxAge returns the minimum max age across all underlying tables.
func (d *DecrementTable) MaxAge() int {
	if d == nil || len(d.tables) == 0 {
		return -1
	}
	maxAge := d.tables[0].MaxAge()
	for _, t := range d.tables[1:] {
		if t.MaxAge() < maxAge {
			maxAge = t.MaxAge()
		}
	}
	return maxAge
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
