// Package reserves calculates policy reserves using prospective and retrospective methods.
//
// # Net premium reserve
//
//	converter := rates.NewRateConverter(0.05)
//	table, _ := mortality.LoadCSV("mortality.csv")
//
//	policy := reserves.PolicySpec{
//	    Age:        30,
//	    Term:       20,
//	    SumAssured: 100000,
//	    Premium:    500,
//	}
//	npr := reserves.NetPremiumReserve(policy, converter, table)
//
// # Gross premium reserve (includes expenses)
//
//	gpr := reserves.GrossPremiumReserve(policy, 50, converter, table) // $50 expense loading
//
// # Prospective reserve (future benefits minus future premiums)
//
//	prosp := reserves.ProspectiveReserve(policy, converter, table)
//
// # Retrospective reserve (accumulated premiums minus past claims)
//
//	retro := reserves.RetrospectiveReserve(policy, converter, table)
//
// # In theory, prospective and retrospective reserves should be equal
// # (equivalence principle). Differences indicate calculation issues.
package reserves
