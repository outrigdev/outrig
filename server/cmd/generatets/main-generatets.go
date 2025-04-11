// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/tsgen"
)

const TypesFileName = "frontend/types/rpctypes.d.ts"
const ClientApiFileName = "frontend/rpc/rpcclientapi.ts"

func generateTypesFile(tsTypesMap map[reflect.Type]string) error {
	fmt.Fprintf(os.Stderr, "generating types file to %s\n", TypesFileName)
	err := tsgen.GenerateRpcServerTypes(tsTypesMap)
	if err != nil {
		return fmt.Errorf("error generating wsh server types: %w", err)
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Copyright 2025, Command Line Inc.\n")
	fmt.Fprintf(&buf, "// SPDX-License-Identifier: Apache-2.0\n\n")
	fmt.Fprintf(&buf, "// generated by cmd/generate/main-generatets.go\n\n")
	fmt.Fprintf(&buf, "declare global {\n\n")
	var keys []reflect.Type
	for key := range tsTypesMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		iname, _ := tsgen.TypeToTSType(keys[i], tsTypesMap)
		jname, _ := tsgen.TypeToTSType(keys[j], tsTypesMap)
		return iname < jname
	})
	for _, key := range keys {
		// don't output generic types
		if strings.Contains(key.Name(), "[") {
			continue
		}
		tsCode := tsTypesMap[key]
		istr := utilfn.IndentString("    ", tsCode)
		fmt.Fprint(&buf, istr)
	}
	fmt.Fprintf(&buf, "}\n\n")
	fmt.Fprintf(&buf, "export {}\n")
	written, err := utilfn.WriteFileIfDifferent(TypesFileName, buf.Bytes())
	if !written {
		fmt.Fprintf(os.Stderr, "no changes to %s\n", TypesFileName)
	}
	return err
}

func generateWshClientApiFile(tsTypeMap map[reflect.Type]string) error {
	var buf bytes.Buffer
	declMap := rpc.GenerateRpcCommandDeclMap()
	fmt.Fprintf(os.Stderr, "generating clientapi file to %s\n", ClientApiFileName)
	fmt.Fprintf(&buf, "// Copyright 2025, Command Line Inc.\n")
	fmt.Fprintf(&buf, "// SPDX-License-Identifier: Apache-2.0\n\n")
	fmt.Fprintf(&buf, "// generated by cmd/generate/main-generatets.go\n\n")
	fmt.Fprintf(&buf, "import { RpcClient } from \"./rpc\";\n\n")
	orderedKeys := utilfn.GetOrderedMapKeys(declMap)
	fmt.Fprintf(&buf, "class RpcApiType {\n")
	for _, methodDecl := range orderedKeys {
		methodDecl := declMap[methodDecl]
		methodStr := tsgen.GenerateRpcClientApiMethod(methodDecl, tsTypeMap)
		fmt.Fprint(&buf, methodStr)
		fmt.Fprintf(&buf, "\n")
	}
	fmt.Fprintf(&buf, "}\n\n")
	fmt.Fprintf(&buf, "export const RpcApi = new RpcApiType();\n")
	written, err := utilfn.WriteFileIfDifferent(ClientApiFileName, buf.Bytes())
	if !written {
		fmt.Fprintf(os.Stderr, "no changes to %s\n", ClientApiFileName)
	}
	return err
}

func main() {
	tsTypesMap := make(map[reflect.Type]string)
	tsgen.GenerateExtraTypes(tsTypesMap)
	err := generateTypesFile(tsTypesMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating types file: %v\n", err)
		os.Exit(1)
	}
	err = generateWshClientApiFile(tsTypesMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating wshserver file: %v\n", err)
		os.Exit(1)
	}
}
