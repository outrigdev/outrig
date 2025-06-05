// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

const MaxTagLen = 40

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

func BoundValue(val, minVal, maxVal int) int {
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

func TryLockWithTimeout(locker sync.Locker, timeout time.Duration) (func(), time.Duration) {
	var totalSleepTime time.Duration

	switch l := locker.(type) {
	case *sync.Mutex:
		if l.TryLock() {
			return l.Unlock, 0
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
				return l.Unlock, totalSleepTime
			}
		}
		return nil, totalSleepTime

	case *sync.RWMutex:
		if l.TryRLock() {
			return l.RUnlock, 0
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
				return l.RUnlock, totalSleepTime
			}
		}
		return nil, totalSleepTime

	default:
		// generic Locker: no timeout available
		startTime := time.Now()
		locker.Lock()
		return locker.Unlock, time.Since(startTime)
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

const SimpleTagRegexStr = `[a-zA-Z0-9][a-zA-Z0-9/_.:-]*`

// must have whitespace or EOL on either side
var TagRegex = regexp.MustCompile(`(?:^|\s)(#` + SimpleTagRegexStr + `)(?:\s|$)`)

// sequence of one-or-more tags (same trailing-ws/EOL rule)
var tagSeqRegex = regexp.MustCompile(`(?:^|\s)(?:#` + SimpleTagRegexStr + `(?:\s|$))+`)

func ParseTags(input string) []string {
	if !strings.Contains(input, "#") {
		return nil
	}
	matches := TagRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1][1:]) // m[1] is "#tag"; drop the '#'
		if len(tag) <= MaxTagLen {
			tags = append(tags, tag)
		}
	}
	return tags
}

func ParseNameAndTags(input string) (string, []string) {
	if !strings.Contains(input, "#") {
		return strings.TrimSpace(input), nil
	}

	matches := TagRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(input), nil
	}
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1][1:])
		if len(tag) <= MaxTagLen {
			tags = append(tags, tag)
		}
	}

	// strip *entire* tag-run (incl. leading & trailing ws) and collapse to one space
	clean := tagSeqRegex.ReplaceAllString(input, " ")
	return strings.TrimSpace(clean), tags
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

func ConvertMap(val any) map[string]any {
	if val == nil {
		return nil
	}
	m, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

// SafeSubstring safely extracts a substring from the original string based on position
func SafeSubstring(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	if start >= len(s) || end <= 0 || start >= end {
		return ""
	}
	return s[start:end]
}

var versionCoreRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

// StripPreReleaseInfo extracts just the version core (major.minor.patch) from a semver string
// Preserves the "v" prefix if present in the original version
func StripPreReleaseInfo(ver string) string {
	return versionCoreRegex.FindString(ver)
}

// IsSemVerCore validates that the whole string *is exactly* a core semver
func IsSemVerCore(ver string) bool {
	return versionCoreRegex.MatchString(ver) && len(ver) == len(versionCoreRegex.FindString(ver))
}

// CompareSemVerCore compares two semver strings and returns -1, 0, or 1 based on just the *core* versions
// It ignores pre-release and build metadata
// Works with "v" prefixed or unprefixed versions
func CompareSemVerCore(ver1, ver2 string) (int, error) {
	ver1 = StripPreReleaseInfo(ver1)
	ver2 = StripPreReleaseInfo(ver2)
	if !IsSemVerCore(ver1) || !IsSemVerCore(ver2) {
		return 0, fmt.Errorf("invalid semver core: %q or %q", ver1, ver2)
	}
	m1 := versionCoreRegex.FindStringSubmatch(ver1)
	m2 := versionCoreRegex.FindStringSubmatch(ver2)
	for i := 1; i <= 3; i++ {
		a, _ := strconv.Atoi(m1[i])
		b, _ := strconv.Atoi(m2[i])
		if a != b {
			if a < b {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

// MakeLaunchUrlCommand creates a command to open a URL in the default browser for the current operating system
func LaunchUrl(url string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("open", url)
		return cmd.Start()
	case "linux":
		cmd := exec.Command("xdg-open", url)
		return cmd.Start()
	case "windows":
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		return cmd.Start()
	default:
		return fmt.Errorf("browser opening not supported on %s", runtime.GOOS)
	}
}

var invalidTagCharRegex = regexp.MustCompile(`[^a-zA-Z0-9/_.:-]`)

func CleanTag(tag string) string {
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.ToLower(tag)
	if len(tag) > MaxTagLen {
		tag = tag[:MaxTagLen]
	}
	tag = invalidTagCharRegex.ReplaceAllString(tag, "_")
	return tag
}

func CleanTagSlice(tags []string) []string {
	var cleanedTags []string
	seen := make(map[string]bool)
	for _, tag := range tags {
		cleaned := CleanTag(tag)
		if cleaned == "" || seen[cleaned] {
			continue
		}
		cleanedTags = append(cleanedTags, cleaned)
		seen[cleaned] = true
	}
	return cleanedTags
}
