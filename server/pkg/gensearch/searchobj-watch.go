// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
)

type WatchSearchObject struct {
	WatchNum int64
	Name     string
	Val      string // Combined value of StrVal, JsonVal, and GoFmtVal
	Str      string // StrVal
	Json     string // JsonVal
	GoFmt    string // GoFmtVal
	Tags     []string
	Type     string

	// Cached values for searches
	NameToLower     string
	ValToLower      string
	StrToLower      string
	JsonToLower     string
	GoFmtToLower    string
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
	if fieldName == "str" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.StrToLower == "" {
				wso.StrToLower = strings.ToLower(wso.Str)
			}
			return wso.StrToLower
		}
		return wso.Str
	}
	if fieldName == "json" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.JsonToLower == "" {
				wso.JsonToLower = strings.ToLower(wso.Json)
			}
			return wso.JsonToLower
		}
		return wso.Json
	}
	if fieldName == "gofmt" {
		if fieldMods&FieldMod_ToLower != 0 {
			if wso.GoFmtToLower == "" {
				wso.GoFmtToLower = strings.ToLower(wso.GoFmt)
			}
			return wso.GoFmtToLower
		}
		return wso.GoFmt
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
		// Combine name, type, and all values with newline delimiters
		if wso.Combined == "" {
			wso.Combined = wso.Name + "\n" + wso.Type + "\n" + wso.Val + "\n" + wso.Str + "\n" + wso.Json + "\n" + wso.GoFmt
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

func makeCombinedVal(w ds.WatchSampleOld) string {
	vals := make([]string, 0, 3) // pre-alloc cap 3
	if w.StrVal != "" {
		vals = append(vals, w.StrVal)
	}
	if w.JsonVal != "" {
		vals = append(vals, w.JsonVal)
	}
	if w.GoFmtVal != "" {
		vals = append(vals, w.GoFmtVal)
	}
	return strings.Join(vals, "\n") // Join([]) == ""
}

// WatchSampleToSearchObject converts a WatchSample to a WatchSearchObject
func WatchSampleToSearchObject(watch ds.WatchSampleOld) SearchObject {
	// Collect non-empty values with pre-allocated capacity
	combinedVal := makeCombinedVal(watch)
	return &WatchSearchObject{
		WatchNum: watch.WatchNum,
		Name:     watch.Name,
		Val:      combinedVal,
		Str:      watch.StrVal,
		Json:     watch.JsonVal,
		GoFmt:    watch.GoFmtVal,
		Tags:     watch.Tags,
		Type:     watch.Type,
	}
}
