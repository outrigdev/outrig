// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"bytes"
)

type LineBuf struct {
	buf        []byte
	inLongLine bool
}

const MaxLineLength = 64 * 1024

func MakeLineBuf() *LineBuf {
	return &LineBuf{
		buf: make([]byte, 0, MaxLineLength),
	}
}

func (lb *LineBuf) GetPartialAndReset() string {
	rtn := string(lb.buf)
	lb.buf = lb.buf[:0]
	lb.inLongLine = false
	return rtn
}

// processes the buffer, returns lines (partial lines are retained)
func (lb *LineBuf) ProcessBuf(readBuf []byte) (lines []string) {
	var pos int
	for pos < len(readBuf) {
		if lb.inLongLine {
			nlIdx := bytes.IndexByte(readBuf[pos:], '\n')
			if nlIdx == -1 {
				return
			}
			pos = pos + nlIdx + 1
			lb.inLongLine = false
			continue
		}
		ch := readBuf[pos]
		pos++
		lb.buf = append(lb.buf, ch)
		if ch == '\n' {
			lines = append(lines, string(lb.buf))
			lb.buf = lb.buf[:0]
			continue
		}
		if len(lb.buf) >= MaxLineLength {
			lines = append(lines, string(lb.buf))
			lb.inLongLine = true
			lb.buf = lb.buf[:0]
			continue
		}
	}
	return
}
