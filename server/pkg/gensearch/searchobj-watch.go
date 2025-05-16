// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strings"
)

type WatchSearchObject struct {
	WatchNum int64
	Name     string
	Val      string // Value of the watch
	Tags     []string
	Type     string

	// Cached values for searches
	NameToLower     string
	ValToLower      string
	TypeToLower     string
	Combined        string
	CombinedToLower string
}

func (wso *WatchSearchObject) GetTags() []string {
	return wso.Tags
}

func (wso *WatchSearchObject) GetId() int64 {
	return wso.WatchNum
}

func (wso *WatchSearchObject) GetField(fieldName string, fieldMods int) string {
	if fieldName == "name" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.NameToLower == "" {
				wso.NameToLower = strings.ToLower(wso.Name)
			}
			return wso.NameToLower
		}
		return wso.Name
	}
	if fieldName == "val" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.ValToLower == "" {
				wso.ValToLower = strings.ToLower(wso.Val)
			}
			return wso.ValToLower
		}
		return wso.Val
	}
	if fieldName == "type" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.TypeToLower == "" {
				wso.TypeToLower = strings.ToLower(wso.Type)
			}
			return wso.TypeToLower
		}
		return wso.Type
	}
	if fieldName == "" {
		// Combine name, type, and value with newline delimiters
		if wso.Combined == "" {
			wso.Combined = wso.Name + "\n" + wso.Type + "\n" + wso.Val
		}

		if fieldMods&FieldMod_ToLower != 0 {
			if wso.CombinedToLower == "" {
				wso.CombinedToLower = strings.ToLower(wso.Combined)
			}
			return wso.CombinedToLower
		}
		return wso.Combined
	}
	return ""
}

