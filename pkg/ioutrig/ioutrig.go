// internal outrig package (used to get around circular references for internal outrig SDK calls)
package ioutrig

var I OutrigInterface

type OutrigInterface interface {
	SetGoRoutineName(name string)
}

type noopOutrigInterface struct{}

func (n noopOutrigInterface) SetGoRoutineName(name string) {
	// No operation
}
