// Package mortality provides mortality tables with survival probability calculations.
//
// # Create a mortality table from qx values
//
//	qx := []float64{0.001, 0.0012, 0.0015, ...} // probability of death at each age
//	table := mortality.NewTable("CSO-80", qx)
//
// # Load from CSV
//
//	table, err := mortality.LoadCSV("mortality.csv")
//	// CSV must have "age" column and either "qx" or "px" column
//
// # Query survival probabilities
//
//	qx := table.Qx(30)       // probability of death between age 30 and 31
//	px := table.Px(30, 20)   // probability of surviving 20 years from age 30
//	ex := table.Ex(65)       // curtate life expectancy at age 65
//	lx := table.Lx(65)       // number surviving to age 65 (from radix 100,000)
//
// # Stream large mortality files
//
//	mortality.StreamCSV("big_mortality.csv", func(age int, qx float64) {
//	    fmt.Printf("Age %d: qx=%.6f\n", age, qx)
//	})
//
// # CSV format
//
// The CSV should have headers with "age" and either "qx" (probability of death)
// or "px" (cumulative survival probability):
//
//	age,qx
//	0,0.001
//	1,0.0008
//	2,0.0006
//	...
package mortality
