// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/rpctypes"
)

type RpcMethodDecl struct {
	Command                 string
	CommandType             string
	MethodName              string
	CommandDataType         reflect.Type
	DefaultResponseDataType reflect.Type
}

var contextRType = reflect.TypeOf((*context.Context)(nil)).Elem()
var wshRpcInterfaceRType = reflect.TypeOf((*rpctypes.FullRpcInterface)(nil)).Elem()

func getRpcCommandType(method reflect.Method) string {
	if method.Type.NumOut() == 1 {
		outType := method.Type.Out(0)
		if outType.Kind() == reflect.Chan {
			return RpcType_ResponseStream
		}
	}
	return RpcType_Call
}

func getRpcMethodResponseType(commandType string, method reflect.Method) reflect.Type {
	switch commandType {
	case RpcType_ResponseStream:
		if method.Type.NumOut() != 1 {
			panic(fmt.Sprintf("method %q has invalid number of return values for response stream", method.Name))
		}
		outType := method.Type.Out(0)
		if outType.Kind() != reflect.Chan {
			panic(fmt.Sprintf("method %q has invalid return type %s for response stream", method.Name, outType))
		}
		elemType := outType.Elem()
		if !strings.HasPrefix(elemType.Name(), "RespUnion") {
			panic(fmt.Sprintf("method %q has invalid return element type %s for response stream (should be RespUnion)", method.Name, elemType))
		}
		respField, found := elemType.FieldByName("Response")
		if !found {
			panic(fmt.Sprintf("method %q has invalid return element type %s for response stream (missing Response field)", method.Name, elemType))
		}
		return respField.Type
	case RpcType_Call:
		if method.Type.NumOut() > 1 {
			return method.Type.Out(0)
		}
		return nil
	default:
		panic(fmt.Sprintf("unsupported command type %q", commandType))
	}
}

func generateRpcCommandDecl(method reflect.Method) *RpcMethodDecl {
	if method.Type.NumIn() == 0 || method.Type.In(0) != contextRType {
		panic(fmt.Sprintf("method %q does not have context as first argument", method.Name))
	}
	cmdStr := method.Name
	decl := &RpcMethodDecl{}
	// remove Command suffix
	if !strings.HasSuffix(cmdStr, "Command") {
		panic(fmt.Sprintf("method %q does not have Command suffix", cmdStr))
	}
	cmdStr = cmdStr[:len(cmdStr)-len("Command")]
	decl.Command = strings.ToLower(cmdStr)
	decl.CommandType = getRpcCommandType(method)
	decl.MethodName = method.Name
	var cdataType reflect.Type
	if method.Type.NumIn() > 1 {
		cdataType = method.Type.In(1)
	}
	decl.CommandDataType = cdataType
	decl.DefaultResponseDataType = getRpcMethodResponseType(decl.CommandType, method)
	return decl
}

func MakeMethodMapForImpl(impl any, declMap map[string]*RpcMethodDecl) map[string]reflect.Method {
	rtype := reflect.TypeOf(impl)
	rtnMap := make(map[string]reflect.Method)
	for midx := 0; midx < rtype.NumMethod(); midx++ {
		method := rtype.Method(midx)
		if !strings.HasSuffix(method.Name, "Command") {
			continue
		}
		commandName := strings.ToLower(method.Name[:len(method.Name)-len("Command")])
		decl := declMap[commandName]
		if decl == nil {
			log.Printf("WARNING: method %q does not match a command method", method.Name)
			continue
		}
		rtnMap[commandName] = method
	}
	return rtnMap

}

func GenerateRpcCommandDeclMap() map[string]*RpcMethodDecl {
	rtype := wshRpcInterfaceRType
	rtnMap := make(map[string]*RpcMethodDecl)
	for midx := 0; midx < rtype.NumMethod(); midx++ {
		method := rtype.Method(midx)
		decl := generateRpcCommandDecl(method)
		rtnMap[decl.Command] = decl
	}
	return rtnMap
}
