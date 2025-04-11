// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/gogen"
	"github.com/outrigdev/outrig/server/pkg/rpc"
)

const RpcClientFileName = "server/pkg/rpcclient/rpcclient.go"

func GenerateRpcClient() error {
	fmt.Fprintf(os.Stderr, "generating rpcclient file to %s\n", RpcClientFileName)
	var buf strings.Builder
	gogen.GenerateBoilerplate(&buf, "rpcclient", []string{
		"github.com/outrigdev/outrig/server/pkg/rpc",
		"github.com/outrigdev/outrig/server/pkg/rpctypes",
	})
	rpcDeclMap := rpc.GenerateRpcCommandDeclMap()
	for _, key := range utilfn.GetOrderedMapKeys(rpcDeclMap) {
		methodDecl := rpcDeclMap[key]
		if methodDecl.CommandType == rpc.RpcType_ResponseStream {
			gogen.GenMethod_ResponseStream(&buf, methodDecl)
		} else if methodDecl.CommandType == rpc.RpcType_Call {
			gogen.GenMethod_Call(&buf, methodDecl)
		} else {
			panic("unsupported command type " + methodDecl.CommandType)
		}
	}
	buf.WriteString("\n")
	written, err := utilfn.WriteFileIfDifferent(RpcClientFileName, []byte(buf.String()))
	if !written {
		fmt.Fprintf(os.Stderr, "no changes to %s\n", RpcClientFileName)
	}
	return err
}

func main() {
	err := GenerateRpcClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating rpcclient: %v\n", err)
		return
	}
}
