// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"bytes"
	"cmp"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var PTLoc *time.Location

func init() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PT", -8*60*60)
	}
	PTLoc = loc
}

func GetHomeDir() string {
	homeVar, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}
	return homeVar
}

func ExpandHomeDir(pathStr string) string {
	if pathStr != "~" && !strings.HasPrefix(pathStr, "~/") && (!strings.HasPrefix(pathStr, `~\`) || runtime.GOOS != "windows") {
		return filepath.Clean(pathStr)
	}
	homeDir := GetHomeDir()
	if pathStr == "~" {
		return homeDir
	}
	expandedPath := filepath.Clean(filepath.Join(homeDir, pathStr[2:]))
	return expandedPath
}

func CopyStrArr(arr []string) []string {
	newArr := make([]string, len(arr))
	copy(newArr, arr)
	return newArr
}

func DrainChan[T any](ch chan T) {
	for range ch {
	}
}

func ReUnmarshal(out any, in any) error {
	barr, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(barr, out)
}

func IndentString(indent string, str string) string {
	splitArr := strings.Split(str, "\n")
	var rtn strings.Builder
	for _, line := range splitArr {
		if line == "" {
			rtn.WriteByte('\n')
			continue
		}
		rtn.WriteString(indent)
		rtn.WriteString(line)
		rtn.WriteByte('\n')
	}
	return rtn.String()
}

func WriteFileIfDifferent(fileName string, contents []byte) (bool, error) {
	oldContents, err := os.ReadFile(fileName)
	if err == nil && bytes.Equal(oldContents, contents) {
		return false, nil
	}
	err = os.WriteFile(fileName, contents, 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func GetOrderedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func GetJsonTag(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return ""
	}
	commaIdx := strings.Index(jsonTag, ",")
	if commaIdx != -1 {
		jsonTag = jsonTag[:commaIdx]
	}
	return jsonTag
}

func SliceIdx[T comparable](arr []T, elem T) int {
	for idx, e := range arr {
		if e == elem {
			return idx
		}
	}
	return -1
}

// removes an element from a slice and modifies the original slice (the backing elements)
// if it removes the last element from the slice, it will return nil so we free the original slice's backing memory
func RemoveElemFromSlice[T comparable](arr []T, elem T) []T {
	idx := SliceIdx(arr, elem)
	if idx == -1 {
		return arr
	}
	if len(arr) == 1 {
		return nil
	}
	return append(arr[:idx], arr[idx+1:]...)
}

func AddElemToSliceUniq[T comparable](arr []T, elem T) []T {
	if SliceIdx(arr, elem) != -1 {
		return arr
	}
	return append(arr, elem)
}

func MoveSliceIdxToFront[T any](arr []T, idx int) []T {
	// create and return a new slice with idx moved to the front
	if idx == 0 || idx >= len(arr) {
		// make a copy still
		return append([]T(nil), arr...)
	}
	rtn := make([]T, 0, len(arr))
	rtn = append(rtn, arr[idx])
	rtn = append(rtn, arr[0:idx]...)
	rtn = append(rtn, arr[idx+1:]...)
	return rtn
}

// matches a delimited string with a pattern string
// the pattern string can contain "*" to match a single part, or "**" to match the rest of the string
// note that "**" may only appear at the end of the string
func StarMatchString(pattern string, s string, delimiter string) bool {
	patternParts := strings.Split(pattern, delimiter)
	stringParts := strings.Split(s, delimiter)
	pLen, sLen := len(patternParts), len(stringParts)

	for i := 0; i < pLen; i++ {
		if patternParts[i] == "**" {
			// '**' must be at the end to be valid
			return i == pLen-1
		}
		if i >= sLen {
			// If string is exhausted but pattern is not
			return false
		}
		if patternParts[i] != "*" && patternParts[i] != stringParts[i] {
			// If current parts don't match and pattern part is not '*'
			return false
		}
	}
	// Check if both pattern and string are fully matched
	return pLen == sLen
}

func BoundValue[T cmp.Ordered](val, minVal, maxVal T) T {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func NeedsLock(rval reflect.Value) bool {
	switch rval.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Struct, reflect.Interface, reflect.Array, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}

var sleepBackoffs = []time.Duration{
	10 * time.Microsecond,
	50 * time.Microsecond,
	100 * time.Microsecond,
	500 * time.Microsecond,
	1 * time.Millisecond,
	2 * time.Millisecond,
	5 * time.Millisecond,
}

func TryLockWithTimeout(locker sync.Locker, timeout time.Duration) (bool, time.Duration) {
	var totalSleepTime time.Duration

	switch l := locker.(type) {
	case *sync.Mutex:
		if l.TryLock() {
			return true, 0
		}
		iter := 0
		for totalSleepTime < timeout {
			sleepTime := sleepBackoffs[len(sleepBackoffs)-1]
			if iter < len(sleepBackoffs) {
				sleepTime = sleepBackoffs[iter]
			}
			iter++
			if totalSleepTime+sleepTime > timeout {
				sleepTime = timeout - totalSleepTime
			}
			time.Sleep(sleepTime)
			totalSleepTime += sleepTime
			if l.TryLock() {
				return true, totalSleepTime
			}
		}
		return false, totalSleepTime

	case *sync.RWMutex:
		if l.TryRLock() {
			return true, 0
		}
		iter := 0
		for totalSleepTime < timeout {
			sleepTime := sleepBackoffs[len(sleepBackoffs)-1]
			if iter < len(sleepBackoffs) {
				sleepTime = sleepBackoffs[iter]
			}
			iter++
			if totalSleepTime+sleepTime > timeout {
				sleepTime = timeout - totalSleepTime
			}
			time.Sleep(sleepTime)
			totalSleepTime += sleepTime
			if l.TryRLock() {
				return true, totalSleepTime
			}
		}
		return false, totalSleepTime

	default:
		// generic Locker: no timeout available
		startTime := time.Now()
		locker.Lock()
		return true, time.Since(startTime)
	}
}

// TeeCopy copies data from src to dst and calls dataCallbackFn with each chunk of data
func TeeCopy(src io.Reader, dst io.Writer, dataCallbackFn func([]byte)) error {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			// Write to destination
			_, werr := dst.Write(buf[:n])
			if werr != nil {
				return werr
			}

			// Call callback if provided
			if dataCallbackFn != nil {
				dataCallbackFn(buf[:n])
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// should match tag regexp in searchparser/parser.go
var tagRegex = regexp.MustCompile(`(?:^|\s)(#[a-zA-Z][a-zA-Z0-9:_.-]+)`)
var tagSeqRegex = regexp.MustCompile(`(?:^|\s)(?:#[a-zA-Z][a-zA-Z0-9:_.-]+)(?:\s+#[a-zA-Z][a-zA-Z0-9:_.-]+)*`)

func ParseTags(input string) []string {
	if !strings.Contains(input, "#") {
		return nil
	}
	matches := tagRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}

	tags := make([]string, len(matches))
	for i, match := range matches {
		tags[i] = strings.ToLower(match[2][1:])
	}
	return tags
}

func ParseNameAndTags(input string) (string, []string) {
	if !strings.Contains(input, "#") {
		return strings.TrimSpace(input), nil
	}
	matches := tagRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(input), nil
	}
	tags := make([]string, len(matches))
	for i, match := range matches {
		// match[1] is the tag (including '#' which we then drop)
		tags[i] = strings.ToLower(match[1][1:])
	}
	// Replace any valid sequence of tags with a single space.
	cleaned := tagSeqRegex.ReplaceAllString(input, " ")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned, tags
}

var goroutineIDRegexp = regexp.MustCompile(`goroutine (\d+)`)

func GetGoroutineID() int64 {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	// Format of the first line of stack trace is "goroutine N [status]:"
	matches := goroutineIDRegexp.FindSubmatch(buf[:n])
	if len(matches) < 2 {
		return -1
	}
	id, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		return -1
	}
	return id
}

// CalculateDeltas converts a slice of values to deltas between consecutive values
// The first value is kept as is, and subsequent values are the difference from the previous value
// If a value is exactly 0, it's treated as a counter reset and outputs 0 (not a negative delta)
func CalculateDeltas(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	deltaValues := make([]float64, len(values))
	deltaValues[0] = values[0] // Keep the first value as is

	// Calculate deltas for the rest
	for i := 1; i < len(values); i++ {
		// If the current value is 0, treat it as a counter reset
		if values[i] == 0 {
			deltaValues[i] = 0
		} else {
			deltaValues[i] = values[i] - values[i-1]
		}
	}

	return deltaValues
}

func ConvertToWallClockPT(t time.Time) time.Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	pstTime := time.Date(year, month, day, hour, min, sec, 0, PTLoc)
	return pstTime
}
