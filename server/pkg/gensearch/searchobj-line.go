// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gensearch

import (
	"strconv"
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

type LogSearchObject struct {
	// Direct line fields
	Msg     string
	Source  string
	LineNum int64

	// Cached values for searches
	MsgToLower    string
	SourceToLower string
	LineNumStr    string
	CachedTags    []string
	TagsParsed    bool
}

// LogLineToSearchObject converts a ds.LogLine to a SearchObject
func LogLineToSearchObject(line ds.LogLine) SearchObject {
	return &LogSearchObject{
		Msg:     line.Msg,
		Source:  line.Source,
		LineNum: line.LineNum,
	}
}

func (lso *LogSearchObject) GetTags() []string {
	if !lso.TagsParsed {
		lso.CachedTags = utilfn.ParseTags(lso.Msg)
		lso.TagsParsed = true
	}
	return lso.CachedTags
}

func (lso *LogSearchObject) GetId() int64 {
	return lso.LineNum
}

func (lso *LogSearchObject) GetField(fieldName string, fieldMods int) string {
	if fieldName == "" || fieldName == "msg" || fieldName == "line" {
		if fieldMods&FieldMod_ToLower != 0 {
			if lso.MsgToLower == "" {
				lso.MsgToLower = strings.ToLower(lso.Msg)
			}
			return lso.MsgToLower
		}
		return lso.Msg
	}
	if fieldName == "source" {
		if fieldMods&FieldMod_ToLower != 0 {
			if lso.SourceToLower == "" {
				lso.SourceToLower = strings.ToLower(lso.Source)
			}
			return lso.SourceToLower
		}
		return lso.Source
	}
	if fieldName == "linenum" {
		if lso.LineNumStr == "" {
			lso.LineNumStr = strconv.FormatInt(lso.LineNum, 10)
		}
		return lso.LineNumStr
	}
	return ""
}
