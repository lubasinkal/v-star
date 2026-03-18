package main

import (
	"fmt"
	"math/rand"
	"os"
)

func main() {
	f, _ := os.Create("10M.csv")
	defer f.Close()
	fmt.Fprintln(f, "age,sex,policy_type,sum_assured,term")
	for range 10000000 {
		age := 18 + rand.Intn(60)
		sex := []string{"male", "female"}[rand.Intn(2)]
		ptype := []string{"term", "whole_life", "endowment"}[rand.Intn(3)]
		sum := float64(10000 + rand.Intn(500000))
		term := 1 + rand.Intn(30)
		fmt.Fprintf(f, "%d,%s,%s,%.2f,%d\n", age, sex, ptype, sum, term)
	}
	fmt.Println("Created 10M.csv with 10,000,000 rows")
}
