package gensearch

import (
	"strconv"
	"strings"
)

type GoRoutineSearchObject struct {
	GoId  int
	Name  string
	Tags  []string
	Stack string

	// Cached values for searches
	NameToLower     string
	GoIdStr         string
	StackToLower    string
	Combined        string
	CombinedToLower string
}

func (gso *GoRoutineSearchObject) GetTags() []string {
	return gso.Tags
}

func (gso *GoRoutineSearchObject) GetField(fieldName string, fieldMods int) string {
	if fieldName == "goid" {
		if gso.GoIdStr == "" {
			gso.GoIdStr = strconv.Itoa(gso.GoId)
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
	if fieldName == "" {
		// Combine name and stack with a newline delimiter
		if gso.Combined == "" {
			gso.Combined = gso.Name + "\n" + gso.Stack
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
