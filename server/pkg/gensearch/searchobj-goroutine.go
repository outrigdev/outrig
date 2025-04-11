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

	// Cached values for searches
	NameToLower     string
	GoIdStr         string
	StackToLower    string
	StateToLower    string
	Combined        string
	CombinedToLower string
}

func (gso *GoRoutineSearchObject) GetTags() []string {
	return gso.Tags
}

func (gso *GoRoutineSearchObject) GetId() int64 {
	return gso.GoId
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
			if gso.NameToLower == "" {
				gso.NameToLower = strings.ToLower(gso.Name)
			}
			return gso.NameToLower
		}
		return gso.Name
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
		// Combine name, state, and stack with a newline delimiter
		if gso.Combined == "" {
			gso.Combined = gso.Name + "\n" + gso.State + "\n" + gso.Stack
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
	return &GoRoutineSearchObject{
		GoId:  gr.GoId,
		Name:  gr.Name,
		Tags:  gr.Tags,
		Stack: gr.RawStackTrace,
		State: gr.RawState,
	}
}
