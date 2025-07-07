// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package runmode

import (
	"go/build"
	"os"
	"strings"

	"github.com/kballard/go-shellquote"
)

// MakeBuildContext creates a build.Context from config args, handling GOFLAGS and -tags
func MakeBuildContext(args []string) *build.Context {
	// Start with the default context which handles GOPATH, GOOS, GOARCH, etc.
	ctx := build.Default

	// Parse -tags from the provided args first - these take priority
	buildTags, foundTags := extractBuildTags(args)
	if foundTags {
		ctx.BuildTags = buildTags
	} else {
		// If no -tags in args, try GOFLAGS environment variable
		if goflags := os.Getenv("GOFLAGS"); goflags != "" {
			// Use shellquote to properly parse GOFLAGS with shell quoting rules
			flagArgs, err := shellquote.Split(goflags)
			if err == nil {
				goflagsTags, foundGoTags := extractBuildTags(flagArgs)
				if foundGoTags {
					ctx.BuildTags = goflagsTags
				}
			}
		}
	}

	return &ctx
}

// parseBuildTags splits a comma-separated tags string and trims whitespace from each tag
func parseBuildTags(tagsValue string) []string {
	if tagsValue == "" {
		return nil
	}
	tags := strings.Split(tagsValue, ",")
	for j := range tags {
		tags[j] = strings.TrimSpace(tags[j])
	}
	return tags
}

// extractBuildTags extracts build tags from -tags arguments in command line args
// Returns the tags and a boolean indicating if -tags was found (even if empty)
func extractBuildTags(args []string) ([]string, bool) {
	var buildTags []string
	foundTags := false
	
	for i := 0; i < len(args); i++ {
		arg := args[i]
		
		if arg == "-tags" && i+1 < len(args) {
			foundTags = true
			tags := parseBuildTags(args[i+1])
			buildTags = append(buildTags, tags...)
			i++ // skip the next argument since we consumed it
		} else if strings.HasPrefix(arg, "-tags=") {
			foundTags = true
			tagsValue := strings.TrimPrefix(arg, "-tags=")
			tags := parseBuildTags(tagsValue)
			buildTags = append(buildTags, tags...)
		}
	}
	
	return buildTags, foundTags
}