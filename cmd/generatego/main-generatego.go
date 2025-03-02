// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/outrigdev/outrig/pkg/gogen"
	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const WshClientFileName = "pkg/rpcclient/rpcclient.go"

func GenerateRpcClient() error {
	fmt.Fprintf(os.Stderr, "generating wshclient file to %s\n", WshClientFileName)
	var buf strings.Builder
	gogen.GenerateBoilerplate(&buf, "wshclient", []string{
		"github.com/outrigdev/outrig/rpc",
	})
	wshDeclMap := rpc.GenerateRpcCommandDeclMap()
	for _, key := range utilfn.GetOrderedMapKeys(wshDeclMap) {
		methodDecl := wshDeclMap[key]
		if methodDecl.CommandType == rpc.RpcType_ResponseStream {
			gogen.GenMethod_ResponseStream(&buf, methodDecl)
		} else if methodDecl.CommandType == rpc.RpcType_Call {
			gogen.GenMethod_Call(&buf, methodDecl)
		} else {
			panic("unsupported command type " + methodDecl.CommandType)
		}
	}
	buf.WriteString("\n")
	written, err := utilfn.WriteFileIfDifferent(WshClientFileName, []byte(buf.String()))
	if !written {
		fmt.Fprintf(os.Stderr, "no changes to %s\n", WshClientFileName)
	}
	return err
}

func main() {
	err := GenerateRpcClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating wshclient: %v\n", err)
		return
	}
}
