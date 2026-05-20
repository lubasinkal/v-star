package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/lubasinkal/v-star/pkg/server"
)

func Serve(args []string) {
	port := "8080"
	for _, arg := range args {
		if strings.HasPrefix(arg, "--port=") {
			if val := strings.SplitN(arg, "=", 2)[1]; val != "" {
				port = val
			}
		}
		if arg == "--help" || arg == "-h" {
			printServeHelp()
			os.Exit(0)
		}
	}

	fmt.Printf("Starting v-star server on http://localhost:%s\n", port)
	fmt.Println("Press Ctrl+C to stop.")

	srv := server.New(":" + port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printServeHelp() {
	fmt.Println(`Usage: v-star serve [--port=8080]

Start the v-star HTTP API server.

Endpoints:
  GET  /health              - Health check
  POST /value               - Calculate present value
  POST /montecarlo          - Run Monte Carlo simulation
  POST /convert-rate        - Convert between nominal/effective rates
  GET  /mortality/{table}   - Get mortality table info
  POST /export/csv          - Export valuation records as CSV
  POST /export/report       - Export valuation as text report
  POST /upload/csv          - Upload CSV file for valuation`)
}
