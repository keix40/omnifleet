package main

import (
	"fmt"
	"os"

	"github.com/keix40/omnifleet/pkg/auth"
)

func main() {
	pwd := os.Getenv("SEED_DEMO_PASSWORD")
	hash, err := auth.HashSeedDemoPassword(pwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed hash: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(hash)
}
