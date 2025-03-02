// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package gogen

import (
	"fmt"
	"strings"

	"github.com/outrigdev/outrig/pkg/rpc"
)

func GenerateBoilerplate(buf *strings.Builder, pkgName string, imports []string) {
	buf.WriteString("// Copyright 2025, Command Line Inc.\n")
	buf.WriteString("// SPDX-License-Identifier: Apache-2.0\n")
	buf.WriteString("\n// Generated Code. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString(fmt.Sprintf("\t%q\n", imp))
		}
		buf.WriteString(")\n\n")
	}
}

func GenMethod_Call(buf *strings.Builder, methodDecl *rpc.RpcMethodDecl) {
	fmt.Fprintf(buf, "// command %q, rpctypes.%s\n", methodDecl.Command, methodDecl.MethodName)
	var dataType string
	dataVarName := "nil"
	if methodDecl.CommandDataType != nil {
		dataType = ", data " + methodDecl.CommandDataType.String()
		dataVarName = "data"
	}
	returnType := "error"
	respName := "_"
	tParamVal := "any"
	if methodDecl.DefaultResponseDataType != nil {
		returnType = "(" + methodDecl.DefaultResponseDataType.String() + ", error)"
		respName = "resp"
		tParamVal = methodDecl.DefaultResponseDataType.String()
	}
	fmt.Fprintf(buf, "func %s(w *rpc.RpcClient%s, opts *rpc.RpcOpts) %s {\n", methodDecl.MethodName, dataType, returnType)
	fmt.Fprintf(buf, "\t%s, err := SendRpcRequestCallHelper[%s](w, %q, %s, opts)\n", respName, tParamVal, methodDecl.Command, dataVarName)
	if methodDecl.DefaultResponseDataType != nil {
		fmt.Fprintf(buf, "\treturn resp, err\n")
	} else {
		fmt.Fprintf(buf, "\treturn err\n")
	}
	fmt.Fprintf(buf, "}\n\n")
}

func GenMethod_ResponseStream(buf *strings.Builder, methodDecl *rpc.RpcMethodDecl) {
	fmt.Fprintf(buf, "// command %q, rpctypes.%s\n", methodDecl.Command, methodDecl.MethodName)
	var dataType string
	dataVarName := "nil"
	if methodDecl.CommandDataType != nil {
		dataType = ", data " + methodDecl.CommandDataType.String()
		dataVarName = "data"
	}
	respType := "any"
	if methodDecl.DefaultResponseDataType != nil {
		respType = methodDecl.DefaultResponseDataType.String()
	}
	fmt.Fprintf(buf, "func %s(w *rpc.RpcClient%s, opts *rpc.RpcOpts) chan rpc.RespUnion[%s] {\n", methodDecl.MethodName, dataType, respType)
	fmt.Fprintf(buf, "\treturn SendRpcRequestResponseStreamHelper[%s](w, %q, %s, opts)\n", respType, methodDecl.Command, dataVarName)
	fmt.Fprintf(buf, "}\n\n")
}
