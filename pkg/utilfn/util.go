package utilfn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
