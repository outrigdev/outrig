// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strconv"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

type GoRoutineSearchObject struct {
	GoId  int64
	Name  string
	Tags  []string
	Stack string
	State string

	// CreatedBy frame data for name formatting
	CreatedByPackage  string
	CreatedByFuncName string
	CreatedByLineNum  int

	// Cached values for searches
	NameToLower          string
	GoIdStr              string
	StackToLower         string
	StateToLower         string
	Combined             string
	CombinedToLower      string
	FormattedName        string
	FormattedNameToLower string
}

func (gso *GoRoutineSearchObject) GetTags() []string {
	return gso.Tags
}

func (gso *GoRoutineSearchObject) GetId() int64 {
	return gso.GoId
}

// cleanFuncName removes parens, asterisks, and .func suffixes from function names
func cleanFuncName(funcname string) string {
	cleaned := strings.ReplaceAll(funcname, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "*", "")
	// Remove .func suffixes like .func1, .func2, etc.
	if idx := strings.Index(cleaned, ".func"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return cleaned
}

// GetName returns the formatted name for the goroutine, matching the frontend formatGoroutineName logic
func (gso *GoRoutineSearchObject) GetName() string {
	if gso.FormattedName != "" {
		return gso.FormattedName
	}

	hasName := gso.Name != ""
	hasCreatedByFrame := gso.CreatedByPackage != "" && gso.CreatedByFuncName != ""

	if !hasCreatedByFrame {
		if hasName {
			gso.FormattedName = gso.Name
		} else {
			gso.FormattedName = "(unnamed)"
		}
		return gso.FormattedName
	}

	// Extract package name (last part after /)
	pkg := gso.CreatedByPackage
	if idx := strings.LastIndex(pkg, "/"); idx != -1 {
		pkg = pkg[idx+1:]
	}

	if hasName {
		gso.FormattedName = "[" + gso.Name + "]"
	} else {
		nameOrFunc := cleanFuncName(gso.CreatedByFuncName)
		if gso.CreatedByLineNum > 0 {
			gso.FormattedName = pkg + "." + nameOrFunc + ":" + strconv.Itoa(gso.CreatedByLineNum)
		} else {
			gso.FormattedName = pkg + "." + nameOrFunc
		}
	}

	return gso.FormattedName
}

func (gso *GoRoutineSearchObject) GetField(fieldName string, fieldMods int) string {
	if fieldName == "goid" {
		if gso.GoIdStr == "" {
			gso.GoIdStr = strconv.FormatInt(gso.GoId, 10)
		}
		return gso.GoIdStr
	}
	if fieldName == "name" {
		if fieldMods&FieldMod_ToLower != 0 {
			if gso.FormattedNameToLower == "" {
				gso.FormattedNameToLower = strings.ToLower(gso.GetName())
			}
			return gso.FormattedNameToLower
		}
		return gso.GetName()
	}
	if fieldName == "stack" {
		if fieldMods&FieldMod_ToLower != 0 {
			if gso.StackToLower == "" {
				gso.StackToLower = strings.ToLower(gso.Stack)
			}
			return gso.StackToLower
		}
		return gso.Stack
	}
	if fieldName == "state" {
		if fieldMods&FieldMod_ToLower != 0 {
			if gso.StateToLower == "" {
				gso.StateToLower = strings.ToLower(gso.State)
			}
			return gso.StateToLower
		}
		return gso.State
	}
	if fieldName == "" {
		// Combine formatted name, state, and stack with a newline delimiter
		if gso.Combined == "" {
			gso.Combined = gso.GetName() + "\n" + gso.State
		}

		if fieldMods&FieldMod_ToLower != 0 {
			if gso.CombinedToLower == "" {
				gso.CombinedToLower = strings.ToLower(gso.Combined)
			}
			return gso.CombinedToLower
		}
		return gso.Combined
	}
	return ""
}

// ParsedGoRoutineToSearchObject converts a ParsedGoRoutine to a GoRoutineSearchObject
func ParsedGoRoutineToSearchObject(gr rpctypes.ParsedGoRoutine) SearchObject {
	gso := &GoRoutineSearchObject{
		GoId:  gr.GoId,
		Name:  gr.Name,
		Tags:  gr.Tags,
		Stack: gr.RawStackTrace,
		State: gr.RawState,
	}

	// Populate CreatedBy frame data if available
	if gr.CreatedByFrame != nil {
		gso.CreatedByPackage = gr.CreatedByFrame.Package
		gso.CreatedByFuncName = gr.CreatedByFrame.FuncName
		gso.CreatedByLineNum = gr.CreatedByFrame.LineNumber
	}

	return gso
}
