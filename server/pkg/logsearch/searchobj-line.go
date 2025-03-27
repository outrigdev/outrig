package logsearch

import (
	"strconv"
	"strings"

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

func (lso *LogSearchObject) GetTags() []string {
	if !lso.TagsParsed {
		_, tags := utilfn.ParseNameAndTags(lso.Msg)
		lso.CachedTags = tags
		lso.TagsParsed = true
	}
	return lso.CachedTags
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
