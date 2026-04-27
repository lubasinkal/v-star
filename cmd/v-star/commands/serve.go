package commands

// TODO: FIX: Add tests for Serve function.
// HINT: Test that server starts and responds to /health endpoint. Use httptest or exec.Command.

// TODO: FIX: Improve port validation and error messages.
// HINT: Validate port is numeric and in valid range. Return clear error if server fails to start.

import (
	"fmt"
	"os"

	"github.com/lubasinkal/v-star/pkg/server"
)

func Serve(args []string) {
	port := "8080"
	for i, arg := range args {
		if arg == "--port" && i+1 < len(args) {
			port = args[i+1]
		}
		if arg == "--help" || arg == "-h" {
			fmt.Println("Usage: v-star serve [--port=8080]")
			fmt.Println("")
			fmt.Println("Start the v-star HTTP API server.")
			fmt.Println("")
			fmt.Println("Endpoints:")
			fmt.Println("  GET  /health              - Health check")
			fmt.Println("  POST /value               - Calculate present value")
			fmt.Println("  POST /montecarlo          - Run Monte Carlo simulation")
			fmt.Println("  POST /convert-rate        - Convert between nominal/effective rates")
			fmt.Println("  GET  /mortality/{table}   - Get mortality table info")
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
