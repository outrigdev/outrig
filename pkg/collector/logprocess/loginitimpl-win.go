//go:build windows

package logprocess

import (
	"fmt"
	"os"
)

func MakeFileWrap(origFile *os.File, source string, callbackFn LogCallbackFnType, shouldBuffer bool) (FileWrap, error) {
	return nil, fmt.Errorf("MakeDupWrap not implemented on windows")
}
