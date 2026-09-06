package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"itcodex/client/internal/client"
)

func main() {
	ctx := context.Background()
	c := client.NewClient("")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "list-collections":
		colls, err := c.ListCollections(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		printJSON(colls)
	case "ping":
		fmt.Println("Client ready, server URL:", c.BaseURL)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ITCodeX Test Client")
	fmt.Println("Usage:")
	fmt.Println("  test-client list-collections  - List all collections")
	fmt.Println("  test-client ping              - Check client configuration")
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}
