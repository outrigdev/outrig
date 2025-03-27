package logsearch

import (
	"strconv"
	"strings"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

type LogSearchObject struct {
	Line          *ds.LogLine
	MsgToLower    string
	SourceToLower string
	LineNumStr    string
	CachedTags    []string
	TagsParsed    bool
}

func (lso *LogSearchObject) GetTags() []string {
	if !lso.TagsParsed {
		_, tags := utilfn.ParseNameAndTags(lso.Line.Msg)
		lso.CachedTags = tags
		lso.TagsParsed = true
	}
	return lso.CachedTags
}

func (lso *LogSearchObject) GetField(fieldName string, fieldMods int) string {
	if fieldName == "" || fieldName == "msg" {
		if fieldMods&FieldMod_ToLower != 0 {
			if lso.MsgToLower == "" {
				lso.MsgToLower = strings.ToLower(lso.Line.Msg)
			}
			return lso.MsgToLower
		}
		return lso.Line.Msg
	}
	if fieldName == "source" {
		if fieldMods&FieldMod_ToLower != 0 {
			if lso.SourceToLower == "" {
				lso.SourceToLower = strings.ToLower(lso.Line.Source)
			}
			return lso.SourceToLower
		}
		return lso.Line.Source
	}
	if fieldName == "linenum" {
		if lso.LineNumStr == "" {
			lso.LineNumStr = strconv.FormatInt(lso.Line.LineNum, 10)
		}
		return lso.LineNumStr
	}
	return ""
}
