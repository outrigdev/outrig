package main

import (
	"fmt"

	"github.com/outrigdev/outrig/server/pkg/updatecheck"
)

func main() {
	appcastVersion, err := updatecheck.GetLatestAppcastRelease()
	if err != nil {
		fmt.Printf("appcast: error - %v\n", err)
	} else {
		fmt.Printf("appcast: %s\n", appcastVersion)
	}

	githubVersion, err := updatecheck.GetLatestRelease()
	if err != nil {
		fmt.Printf("github-release: error - %v\n", err)
	} else {
		fmt.Printf("github-release: %s\n", githubVersion)
	}
}
