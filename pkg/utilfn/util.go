package utilfn

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

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

func GoDrainChan[T any](ch chan T) {
	go func() {
		for range ch {
		}
	}()
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
