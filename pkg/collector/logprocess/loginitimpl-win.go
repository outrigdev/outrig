//go:build windows

package logprocess

import (
	"fmt"
	"os"
)

func MakeFileWrap(origFile *os.File, source string, callbackFn LogCallbackFnType) (FileWrap, error) {
	return nil, fmt.Errorf("MakeDupWrap not implemented on windows")
}
