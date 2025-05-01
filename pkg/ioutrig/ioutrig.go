// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// internal outrig package (used to get around circular references for internal outrig SDK calls)
package ioutrig

var I OutrigInterface = noopOutrigInterface{}

type OutrigInterface interface {
	SetGoRoutineName(name string)
	Log(str string)
	Logf(format string, args ...any)
}

type noopOutrigInterface struct{}

func (n noopOutrigInterface) SetGoRoutineName(name string) {
	// No operation
}

func (n noopOutrigInterface) Log(str string) {
	// No operation
}

func (n noopOutrigInterface) Logf(format string, args ...any) {
	// No operation
}
