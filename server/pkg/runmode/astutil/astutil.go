// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package astutil

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/outrigdev/outrig/pkg/config"
	"github.com/outrigdev/outrig/server/pkg/serverbase"
	"golang.org/x/mod/modfile"
)

const outrigCommentPrefix = "//outrig"

const ScopeGo = "go"

// outrigDirectiveRegex matches lines that start with //outrig optionally followed by :scope and optionally key="value" pairs
var outrigDirectiveRegex = regexp.MustCompile(`^//outrig(?::(\w+))?(?:\s+(\w+="[^"]*"\s*)+)?$`)

// outrigCommentRegex matches valid outrig comments: //outrig, //outrig:scope, //outrig args, //outrig:scope args
var outrigCommentRegex = regexp.MustCompile(`^//outrig(?::|$|\s)`)

// keyValueRegex extracts key="value" pairs from the directive
var keyValueRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)

// OutrigDirective represents a parsed //outrig comment directive
type OutrigDirective struct {
	Go OutrigGoDirective
}

type OutrigGoDirective struct {
	Name string
	Tags string
}

const OutrigImportPath = "github.com/outrigdev/outrig"

// HasImport checks if the given import path exists in the AST node
func HasImport(node *ast.File, importPath string) bool {
	// Check node.Imports (populated during parsing)
	for _, imp := range node.Imports {
		if imp.Path.Value == `"`+importPath+`"` {
			return true
		}
	}

	// Also check import declarations in node.Decls (for programmatically added imports)
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					if importSpec.Path.Value == `"`+importPath+`"` {
						return true
					}
				}
			}
		}
	}

	return false
}

func AddOutrigImportReplacement(state *TransformState, file *ModifiedFile) error {
	// Check if outrig import already added via replacement
	if file.OutrigImportAdded {
		return nil
	}

	// Check if outrig import already exists
	if HasImport(file.FileAST, OutrigImportPath) {
		return nil
	}

	// Find the position after the package declaration
	packagePos := file.FileAST.Name.End()

	// Convert token position to file position
	position := state.FileSet.Position(packagePos)

	// Add the import statement using the new AddInsertStmt method
	importText := "import \"" + OutrigImportPath + "\""
	file.AddInsertStmt(position, importText)

	// Mark that we've added the outrig import
	file.OutrigImportAdded = true

	return nil
}

// AddOutrigImport checks if the outrig import exists in the AST node and adds it if not present.
// Returns true if the import was added, false if it already existed.
func AddOutrigImport(fset *token.FileSet, node *ast.File) bool {
	// Check if outrig import already exists
	if HasImport(node, OutrigImportPath) {
		return false
	}

	// Add outrig import since it's not present
	outrigImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + OutrigImportPath + `"`,
		},
	}

	// Create a new import declaration for outrig
	importDecl := &ast.GenDecl{
		Tok:   token.IMPORT,
		Specs: []ast.Spec{outrigImport},
	}

	// Find the position to insert the new import (after existing imports if any)
	insertPos := 0
	for i, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			insertPos = i + 1
		} else {
			break
		}
	}

	// Insert the new import declaration
	node.Decls = append(node.Decls[:insertPos], append([]ast.Decl{importDecl}, node.Decls[insertPos:]...)...)
	return true
}

// AstFileWrap represents a Go file that has been processed with AST transformations
type AstFileWrap struct {
	OriginalPath string
	ModifiedAST  *ast.File
	FileSet      *token.FileSet
	WasModified  bool
}

// WriteToTempFile writes the AST to a temporary file in the specified directory
func (a *AstFileWrap) WriteToTempFile(tempDir string) (string, error) {
	tempFileName := GenerateTempFileName(a.OriginalPath)
	tempFilePath := filepath.Join(tempDir, tempFileName)

	return tempFilePath, WriteASTToFile(a.FileSet, a.ModifiedAST, tempFilePath)
}

// FindMainFunction returns the main function declaration if it exists with proper signature in package main, nil otherwise
func FindMainFunction(node *ast.File) *ast.FuncDecl {
	// Check that this is package main
	if node.Name.Name != "main" {
		return nil
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" {
			continue
		}

		// Check that it's not a method (no receiver)
		if fn.Recv != nil {
			continue
		}

		// Check that it has no parameters
		if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
			continue
		}

		// Check that it has no return values
		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			continue
		}

		return fn
	}
	return nil
}

// WriteASTToFile writes an AST node to a file using the provided file set
func WriteASTToFile(fset *token.FileSet, node *ast.File, fileName string) error {
	var buf strings.Builder
	config := &printer.Config{
		Mode: printer.SourcePos, // Generate line directives to preserve original line numbers
	}
	err := config.Fprint(&buf, fset, node)
	if err != nil {
		return fmt.Errorf("failed to print modified code: %w", err)
	}

	err = os.WriteFile(fileName, []byte(buf.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fileName, err)
	}

	return nil
}

// GenerateTempFileName creates a unique filename for the temp directory by hashing the original path
func GenerateTempFileName(originalPath string) string {
	// Get the base filename
	baseName := filepath.Base(originalPath)

	// Remove .go extension
	nameWithoutExt := strings.TrimSuffix(baseName, ".go")

	// Create hash of the full original path to avoid conflicts
	hash := sha256.Sum256([]byte(originalPath))
	hashStr := fmt.Sprintf("%x", hash)[:8] // Use first 8 chars of hash

	// Return formatted filename
	return fmt.Sprintf("%s_%s.go", nameWithoutExt, hashStr)
}

// parseOutrigScopeFromComment extracts the scope from an outrig comment (e.g., "outrig:go" returns "go")
func parseOutrigScopeFromComment(content string) string {
	if !strings.HasPrefix(content, "outrig:") {
		return ""
	}

	scopeStart := 7 // len("outrig:")
	spaceIdx := strings.Index(content[scopeStart:], " ")
	if spaceIdx == -1 {
		return content[scopeStart:]
	}
	return content[scopeStart : scopeStart+spaceIdx]
}

// parseOutrigDirective parses an //outrig comment and merges the directive information into existing directive
func parseOutrigDirective(comment string, scope string, existing *OutrigDirective) (*OutrigDirective, error) {
	// Remove leading // and whitespace
	content := strings.TrimSpace(strings.TrimPrefix(comment, "//"))

	// Parse scope from comment if present (e.g., "outrig:go name=..." or "outrig name=...")
	commentScope := parseOutrigScopeFromComment(content)
	if commentScope == "" {
		commentScope = scope
	}

	// Only process "go" scope for now
	if commentScope != ScopeGo {
		return existing, nil
	}

	directive := existing
	if directive == nil {
		directive = &OutrigDirective{}
	}

	// Extract key-value pairs
	matches := keyValueRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		key := match[1]
		value := match[2]

		switch key {
		case "name":
			directive.Go.Name = value
		case "tags":
			directive.Go.Tags = value
		}
	}

	return directive, nil
}

// ParseOutrigDirective looks for //outrig comments in the comment group and returns the combined directive
func ParseOutrigDirective(comments []*ast.CommentGroup, scope string) OutrigDirective {
	if len(comments) == 0 {
		return OutrigDirective{}
	}

	var combinedDirective *OutrigDirective

	for _, commentGroup := range comments {
		for _, comment := range commentGroup.List {
			if !outrigCommentRegex.MatchString(comment.Text) {
				continue
			}

			directive, err := parseOutrigDirective(comment.Text, scope, combinedDirective)
			if err != nil {
				// Skip invalid directives but don't fail the build
				continue
			}
			combinedDirective = directive
		}
	}

	if combinedDirective == nil {
		return OutrigDirective{}
	}

	return *combinedDirective
}

// findLeadingComments finds the comment group that appears immediately before the given statement
func findLeadingComments(fset *token.FileSet, file *ast.File, targetStmt ast.Stmt) *ast.CommentGroup {
	cmap := ast.NewCommentMap(fset, file, file.Comments)
	stmtPos := targetStmt.Pos()

	// Get all comment groups associated with this statement
	comments := cmap[targetStmt]

	// Find the comment group that ends before the statement starts
	for _, commentGroup := range comments {
		if commentGroup.End() < stmtPos {
			return commentGroup
		}
	}

	return nil
}

// ParseOutrigDirectiveForStmt looks for //outrig comments immediately before the given statement
func ParseOutrigDirectiveForStmt(fset *token.FileSet, file *ast.File, stmt ast.Stmt, scope string) OutrigDirective {
	commentGroup := findLeadingComments(fset, file, stmt)
	if commentGroup == nil {
		return OutrigDirective{}
	}

	return ParseOutrigDirective([]*ast.CommentGroup{commentGroup}, scope)
}

// MakeLineDirective creates a //line directive string for the given file path and line number
func MakeLineDirective(filePath string, lineNum int) string {
	return fmt.Sprintf("//line %s:%d\n", filePath, lineNum)
}

// getExistingOutrigSDKVersion returns the version of the outrig SDK if it exists in the go.mod file, empty string otherwise
func getExistingOutrigSDKVersion(goModFile *modfile.File) string {
	for _, req := range goModFile.Require {
		if req.Mod.Path == "github.com/outrigdev/outrig" {
			return req.Mod.Version
		}
	}
	return ""
}

// validateSDKVersionCompatibility validates that the SDK version is compatible with the server
// Requires exact major/minor version match between SDK and server
// Returns nil if sdkVersion is empty string
func validateSDKVersionCompatibility(sdkVersion string) error {
	if sdkVersion == "" {
		return nil
	}

	serverVersion := serverbase.OutrigServerVersion

	existingVer, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return fmt.Errorf("invalid SDK version format %s: %w", sdkVersion, err)
	}

	serverVer, err := semver.NewVersion(serverVersion)
	if err != nil {
		return fmt.Errorf("invalid server version format %s: %w", serverVersion, err)
	}

	// Require exact major and minor version match
	if existingVer.Major() != serverVer.Major() || existingVer.Minor() != serverVer.Minor() {
		return fmt.Errorf("SDK version %s major/minor does not match server version %s", sdkVersion, serverVersion)
	}

	return nil
}

// AddOutrigSDKDependency adds the version locked Outrig SDK to the temp go.mod file
func AddOutrigSDKDependency(tempGoModPath string, verbose bool) error {
	// Read and parse the current go.mod file
	goModData, err := os.ReadFile(tempGoModPath)
	if err != nil {
		return fmt.Errorf("failed to read temp go.mod: %w", err)
	}

	goModFile, err := modfile.Parse(tempGoModPath, goModData, nil)
	if err != nil {
		return fmt.Errorf("failed to parse temp go.mod: %w", err)
	}

	// Check if the outrig SDK dependency already exists and validate version
	existingVersion := getExistingOutrigSDKVersion(goModFile)
	if err := validateSDKVersionCompatibility(existingVersion); err != nil {
		return fmt.Errorf("incompatible SDK version %s: %w", existingVersion, err)
	}

	// Add the Outrig SDK dependency with the specified version
	err = goModFile.AddRequire("github.com/outrigdev/outrig", config.OutrigSDKVersion)
	if err != nil {
		return fmt.Errorf("failed to add outrig SDK dependency: %w", err)
	}

	if verbose {
		log.Printf("Adding outrig SDK dependency: github.com/outrigdev/outrig %s", config.OutrigSDKVersion)
	}

	// Format and write the modified go.mod file
	formattedData, err := goModFile.Format()
	if err != nil {
		return fmt.Errorf("failed to format modified go.mod: %w", err)
	}

	err = os.WriteFile(tempGoModPath, formattedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified go.mod: %w", err)
	}

	if verbose {
		log.Printf("Added outrig SDK dependency to temp go.mod")
	}

	return nil
}
