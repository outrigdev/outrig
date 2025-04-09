// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	moduleRoot := flag.String("root", ".", "module root directory")
	modulePrefix := flag.String("module", "", "module prefix (e.g. github.com/outrigdev/outrig)")
	flag.Parse()

	if *modulePrefix == "" {
		fmt.Println("Module prefix is required (use -module)")
		os.Exit(1)
	}

	// Build graph: pkg -> set(imported pkgs)
	graph := make(map[string]map[string]bool)
	// Keep track of every package we see
	packages := make(map[string]bool)

	err := filepath.Walk(*moduleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Only process .go files.
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		// Determine the package relative path.
		pkgDir := filepath.Dir(path)
		relPkg, err := filepath.Rel(*moduleRoot, pkgDir)
		if err != nil {
			return err
		}
		relPkg = filepath.ToSlash(relPkg)
		packages[relPkg] = true

		if graph[relPkg] == nil {
			graph[relPkg] = make(map[string]bool)
		}

		// Process imports.
		for _, imp := range f.Imports {
			importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove quotes
			// Only consider packages in the module.
			if !strings.HasPrefix(importPath, *modulePrefix+"/") {
				continue
			}
			localImport := strings.TrimPrefix(importPath, *modulePrefix+"/")
			// Check if the package exists on disk.
			if _, err := os.Stat(filepath.Join(*moduleRoot, filepath.FromSlash(localImport))); err != nil {
				continue
			}
			packages[localImport] = true
			graph[relPkg][localImport] = true
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error scanning module:", err)
		os.Exit(1)
	}

	// Ensure every found package is in the graph.
	for pkg := range packages {
		if _, ok := graph[pkg]; !ok {
			graph[pkg] = make(map[string]bool)
		}
	}

	// Compute incoming edge counts.
	incoming := make(map[string]int)
	for pkg := range packages {
		incoming[pkg] = 0
	}
	for _, deps := range graph {
		for dep := range deps {
			incoming[dep]++
		}
	}

	// Topologically compute levels: nodes with zero incoming edges are level 0.
	levels := make(map[string]int)
	queue := []string{}
	for pkg, count := range incoming {
		if count == 0 {
			queue = append(queue, pkg)
			levels[pkg] = 0
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for dep := range graph[current] {
			if levels[dep] < levels[current]+1 {
				levels[dep] = levels[current] + 1
			}
			incoming[dep]--
			if incoming[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Output DOT with rank grouping per level.
	fmt.Println("digraph G {")
	fmt.Println("  rankdir=TB;")
	// Group packages by level.
	levelGroups := make(map[int][]string)
	for pkg, lvl := range levels {
		levelGroups[lvl] = append(levelGroups[lvl], pkg)
	}
	for lvl, nodes := range levelGroups {
		fmt.Printf("  { rank = same; ")
		for _, n := range nodes {
			fmt.Printf("\"%s\"; ", n)
		}
		fmt.Printf("} // level %d\n", lvl)
	}
	// Output dependency edges.
	for from, deps := range graph {
		for dep := range deps {
			if _, ok := levels[from]; ok {
				if _, ok2 := levels[dep]; ok2 {
					fmt.Printf("  \"%s\" -> \"%s\";\n", from, dep)
				}
			}
		}
	}
	fmt.Println("}")
}
