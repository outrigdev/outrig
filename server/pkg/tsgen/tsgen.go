// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package tsgen

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/outrigdev/outrig/pkg/rpc"
	"github.com/outrigdev/outrig/pkg/rpctypes"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/tsgen/tsgenmeta"
)

// add extra types to generate here
var ExtraTypes = []any{
	map[string]any{},
	rpc.RpcMessage{},
	rpctypes.ServerCommandMeta{},
	rpctypes.EventCommonFields{},
}

// add extra type unions to generate here
var TypeUnions = []tsgenmeta.TypeUnionMeta{}

var contextRType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorRType = reflect.TypeOf((*error)(nil)).Elem()
var anyRType = reflect.TypeOf((*interface{})(nil)).Elem()
var eventTypeRType = reflect.TypeOf((*rpctypes.EventType)(nil)).Elem()
var rpcInterfaceRType = reflect.TypeOf((*rpctypes.FullRpcInterface)(nil)).Elem()

func generateTSMethodTypes(method reflect.Method, tsTypesMap map[reflect.Type]string, skipFirstArg bool) error {
	for idx := 0; idx < method.Type.NumIn(); idx++ {
		if skipFirstArg && idx == 0 {
			continue
		}
		inType := method.Type.In(idx)
		GenerateTSType(inType, tsTypesMap)
	}
	for idx := 0; idx < method.Type.NumOut(); idx++ {
		outType := method.Type.Out(idx)
		GenerateTSType(outType, tsTypesMap)
	}
	return nil
}

func getTSFieldName(field reflect.StructField) string {
	tsFieldTag := field.Tag.Get("tsfield")
	if tsFieldTag != "" {
		if tsFieldTag == "-" {
			return ""
		}
		return tsFieldTag
	}
	jsonTag := utilfn.GetJsonTag(field)
	if jsonTag == "-" {
		return ""
	}
	if strings.Contains(jsonTag, ":") {
		return "\"" + jsonTag + "\""
	}
	if jsonTag != "" {
		return jsonTag
	}
	return field.Name
}

func isFieldOmitEmpty(field reflect.StructField) bool {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 1 {
			for _, part := range parts[1:] {
				if part == "omitempty" {
					return true
				}
			}
		}
	}
	return false
}

func TypeToTSType(t reflect.Type, tsTypesMap map[reflect.Type]string) (string, []reflect.Type) {
	switch t.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Slice, reflect.Array:
		// special case for byte slice, marshals to base64 encoded string
		if t.Elem().Kind() == reflect.Uint8 {
			return "string", nil
		}
		elemType, subTypes := TypeToTSType(t.Elem(), tsTypesMap)
		if elemType == "" {
			return "", nil
		}
		return fmt.Sprintf("%s[]", elemType), subTypes
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return "", nil
		}
		elemType, subTypes := TypeToTSType(t.Elem(), tsTypesMap)
		if elemType == "" {
			return "", nil
		}
		return fmt.Sprintf("{[key: string]: %s}", elemType), subTypes
	case reflect.Struct:
		name := t.Name()
		if tsRename := tsRenameMap[name]; tsRename != "" {
			name = tsRename
		}
		return name, []reflect.Type{t}
	case reflect.Ptr:
		return TypeToTSType(t.Elem(), tsTypesMap)
	case reflect.Interface:
		if _, ok := tsTypesMap[t]; ok {
			return t.Name(), nil
		}
		return "any", nil
	default:
		return "", nil
	}
}

var tsRenameMap = map[string]string{}

func generateEventType(tsTypesMap map[reflect.Type]string) (string, []reflect.Type) {
	var buf bytes.Buffer
	var extraTypes []reflect.Type

	buf.WriteString("// EventType union (rpctypes.EventToTypeMap)\n")
	buf.WriteString("type EventType = \n")
	tmap := rpctypes.EventToTypeMap

	// Extract and sort keys for deterministic output
	eventNames := make([]string, 0, len(tmap))
	for eventName := range tmap {
		eventNames = append(eventNames, eventName)
	}
	sort.Strings(eventNames)
	for _, eventName := range eventNames {
		rtype := tmap[eventName]
		var tsType string
		var optStr string
		if rtype != nil {
			extraTypes = append(extraTypes, rtype)
			tsType, _ = TypeToTSType(rtype, tsTypesMap)
		} else {
			tsType = "null"
			optStr = "?"
		}
		buf.WriteString(fmt.Sprintf("    | (EventCommonFields & { event: %q; data%s: %s })\n", eventName, optStr, tsType))
	}
	buf.WriteString(";\n")
	return buf.String(), extraTypes
}

func generateTSTypeInternal(rtype reflect.Type, tsTypesMap map[reflect.Type]string, embedded bool) (string, []reflect.Type) {
	if rtype == eventTypeRType {
		return generateEventType(tsTypesMap)
	}
	var buf bytes.Buffer
	tsTypeName := rtype.Name()
	if tsRename, ok := tsRenameMap[tsTypeName]; ok {
		tsTypeName = tsRename
	}
	if !embedded {
		buf.WriteString(fmt.Sprintf("// %s\n", rtype.String()))
		buf.WriteString(fmt.Sprintf("type %s = {\n", tsTypeName))
	}
	var subTypes []reflect.Type
	for i := 0; i < rtype.NumField(); i++ {
		field := rtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			embeddedBuf, embeddedTypes := generateTSTypeInternal(field.Type, tsTypesMap, true)
			buf.WriteString(embeddedBuf)
			subTypes = append(subTypes, embeddedTypes...)
			continue
		}
		fieldName := getTSFieldName(field)
		if fieldName == "" {
			continue
		}
		optMarker := ""
		if isFieldOmitEmpty(field) {
			optMarker = "?"
		}
		tsTypeTag := field.Tag.Get("tstype")
		if tsTypeTag != "" {
			if tsTypeTag == "-" {
				continue
			}
			buf.WriteString(fmt.Sprintf("    %s%s: %s;\n", fieldName, optMarker, tsTypeTag))
			continue
		}
		tsType, fieldSubTypes := TypeToTSType(field.Type, tsTypesMap)
		if tsType == "" {
			continue
		}
		subTypes = append(subTypes, fieldSubTypes...)
		if tsType == "UIContext" {
			optMarker = "?"
		}
		buf.WriteString(fmt.Sprintf("    %s%s: %s;\n", fieldName, optMarker, tsType))
	}
	if !embedded {
		buf.WriteString("};\n")
	}
	return buf.String(), subTypes
}

func GenerateTSTypeUnion(unionMeta tsgenmeta.TypeUnionMeta, tsTypeMap map[reflect.Type]string) {
	rtn := generateTSTypeUnionInternal(unionMeta)
	tsTypeMap[unionMeta.BaseType] = rtn
	for _, rtype := range unionMeta.Types {
		GenerateTSType(rtype, tsTypeMap)
	}
}

func generateTSTypeUnionInternal(unionMeta tsgenmeta.TypeUnionMeta) string {
	var buf bytes.Buffer
	if unionMeta.Desc != "" {
		buf.WriteString(fmt.Sprintf("// %s\n", unionMeta.Desc))
	}
	buf.WriteString(fmt.Sprintf("type %s = {\n", unionMeta.BaseType.Name()))
	buf.WriteString(fmt.Sprintf("    %s: string;\n", unionMeta.TypeFieldName))
	buf.WriteString("} & ( ")
	for idx, rtype := range unionMeta.Types {
		if idx > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(rtype.Name())
	}
	buf.WriteString(" );\n")
	return buf.String()
}

func GenerateTSType(rtype reflect.Type, tsTypesMap map[reflect.Type]string) {
	if rtype == nil {
		return
	}
	if rtype.Kind() == reflect.Chan {
		rtype = rtype.Elem()
	}
	if rtype == contextRType || rtype == errorRType || rtype == anyRType {
		return
	}
	if rtype.Kind() == reflect.Slice {
		rtype = rtype.Elem()
	}
	if rtype.Kind() == reflect.Map {
		rtype = rtype.Elem()
	}
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	if _, ok := tsTypesMap[rtype]; ok {
		return
	}
	if rtype.Kind() != reflect.Struct {
		return
	}
	tsType, subTypes := generateTSTypeInternal(rtype, tsTypesMap, false)
	tsTypesMap[rtype] = tsType
	for _, subType := range subTypes {
		GenerateTSType(subType, tsTypesMap)
	}
}

func GenerateMethodSignature(serviceName string, method reflect.Method, meta tsgenmeta.MethodMeta, isFirst bool, tsTypesMap map[reflect.Type]string) string {
	var sb strings.Builder
	if (meta.Desc != "" || meta.ReturnDesc != "") && !isFirst {
		sb.WriteString("\n")
	}
	if meta.Desc != "" {
		sb.WriteString(fmt.Sprintf("    // %s\n", meta.Desc))
	}
	if meta.ReturnDesc != "" {
		sb.WriteString(fmt.Sprintf("    // @returns %s\n", meta.ReturnDesc))
	}
	sb.WriteString("    ")
	sb.WriteString(method.Name)
	sb.WriteString("(")
	wroteArg := false
	// skip first arg, which is the receiver
	for idx := 1; idx < method.Type.NumIn(); idx++ {
		if wroteArg {
			sb.WriteString(", ")
		}
		inType := method.Type.In(idx)
		if inType == contextRType {
			continue
		}
		tsTypeName, _ := TypeToTSType(inType, tsTypesMap)
		var argName string
		if idx-1 < len(meta.ArgNames) {
			argName = meta.ArgNames[idx-1] // subtract 1 for receiver
		} else {
			argName = fmt.Sprintf("arg%d", idx)
		}
		sb.WriteString(fmt.Sprintf("%s: %s", argName, tsTypeName))
		wroteArg = true
	}
	sb.WriteString("): ")
	rtnTypes := []string{}
	for idx := 0; idx < method.Type.NumOut(); idx++ {
		outType := method.Type.Out(idx)
		if outType == errorRType {
			continue
		}
		tsTypeName, _ := TypeToTSType(outType, tsTypesMap)
		rtnTypes = append(rtnTypes, tsTypeName)
	}
	if len(rtnTypes) == 0 {
		sb.WriteString("Promise<void>")
	} else if len(rtnTypes) == 1 {
		sb.WriteString(fmt.Sprintf("Promise<%s>", rtnTypes[0]))
	} else {
		sb.WriteString(fmt.Sprintf("Promise<[%s]>", strings.Join(rtnTypes, ", ")))
	}
	sb.WriteString(" {\n")
	return sb.String()
}

func GenerateMethodBody(serviceName string, method reflect.Method, meta tsgenmeta.MethodMeta) string {
	return fmt.Sprintf("        return WOS.callBackendService(%q, %q, Array.from(arguments))\n", serviceName, method.Name)
}

func GenerateRpcClientApiMethod(methodDecl *rpc.RpcMethodDecl, tsTypesMap map[reflect.Type]string) string {
	if methodDecl.CommandType == rpc.RpcType_ResponseStream {
		return generateRpcClientApiMethod_ResponseStream(methodDecl, tsTypesMap)
	} else if methodDecl.CommandType == rpc.RpcType_Call {
		return generateRpcClientApiMethod_Call(methodDecl, tsTypesMap)
	} else {
		panic(fmt.Sprintf("cannot generate rpcserver commandtype %q", methodDecl.CommandType))
	}
}

func generateRpcClientApiMethod_ResponseStream(methodDecl *rpc.RpcMethodDecl, tsTypesMap map[reflect.Type]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("    // command %q [%s]\n", methodDecl.Command, methodDecl.CommandType))
	respType := "any"
	if methodDecl.DefaultResponseDataType != nil {
		respType, _ = TypeToTSType(methodDecl.DefaultResponseDataType, tsTypesMap)
	}
	dataName := "null"
	if methodDecl.CommandDataType != nil {
		dataName = "data"
	}
	genRespType := fmt.Sprintf("AsyncGenerator<%s, void, boolean>", respType)
	if methodDecl.CommandDataType != nil {
		cmdDataTsName, _ := TypeToTSType(methodDecl.CommandDataType, tsTypesMap)
		sb.WriteString(fmt.Sprintf("	%s(client: RpcClient, data: %s, opts?: RpcOpts): %s {\n", methodDecl.MethodName, cmdDataTsName, genRespType))
	} else {
		sb.WriteString(fmt.Sprintf("	%s(client: RpcClient, opts?: RpcOpts): %s {\n", methodDecl.MethodName, genRespType))
	}
	sb.WriteString(fmt.Sprintf("        return client.rpcStream(%q, %s, opts);\n", methodDecl.Command, dataName))
	sb.WriteString("    }\n")
	return sb.String()
}

func generateRpcClientApiMethod_Call(methodDecl *rpc.RpcMethodDecl, tsTypesMap map[reflect.Type]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("    // command %q [%s]\n", methodDecl.Command, methodDecl.CommandType))
	rtnType := "Promise<void>"
	if methodDecl.DefaultResponseDataType != nil {
		rtnTypeName, _ := TypeToTSType(methodDecl.DefaultResponseDataType, tsTypesMap)
		rtnType = fmt.Sprintf("Promise<%s>", rtnTypeName)
	}
	dataName := "null"
	if methodDecl.CommandDataType != nil {
		dataName = "data"
	}
	if methodDecl.CommandDataType != nil {
		cmdDataTsName, _ := TypeToTSType(methodDecl.CommandDataType, tsTypesMap)
		sb.WriteString(fmt.Sprintf("    %s(client: RpcClient, data: %s, opts?: RpcOpts): %s {\n", methodDecl.MethodName, cmdDataTsName, rtnType))
	} else {
		sb.WriteString(fmt.Sprintf("    %s(client: RpcClient, opts?: RpcOpts): %s {\n", methodDecl.MethodName, rtnType))
	}
	methodBody := fmt.Sprintf("        return client.rpcCall(%q, %s, opts);\n", methodDecl.Command, dataName)
	sb.WriteString(methodBody)
	sb.WriteString("    }\n")
	return sb.String()
}

func GenerateRpcServerTypes(tsTypesMap map[reflect.Type]string) error {
	GenerateTSType(reflect.TypeOf(rpc.RpcOpts{}), tsTypesMap)
	rtype := rpcInterfaceRType
	for midx := 0; midx < rtype.NumMethod(); midx++ {
		method := rtype.Method(midx)
		err := generateTSMethodTypes(method, tsTypesMap, false)
		if err != nil {
			return fmt.Errorf("error generating TS method types for %s.%s: %v", rtype, method.Name, err)
		}
	}
	return nil
}

func GenerateExtraTypes(tsTypesMap map[reflect.Type]string) {
	for _, extraType := range ExtraTypes {
		GenerateTSType(reflect.TypeOf(extraType), tsTypesMap)
	}
}
