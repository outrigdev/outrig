package main

import (
	"fmt"
	"log"

	"github.com/outrigdev/outrig/server/pkg/updatecheck"
)

func main() {
	version, err := updatecheck.GetLatestAppcastRelease()
	if err != nil {
		log.Fatalf("Error getting latest appcast release: %v", err)
	}
	
	fmt.Printf("%s\n", version)
}