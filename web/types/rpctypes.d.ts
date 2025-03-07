// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// generated by cmd/generate/main-generatets.go

declare global {

    // rpctypes.CommandMessageData
    type CommandMessageData = {
        message: string;
    };

    // rpctypes.DropRequestData
    type DropRequestData = {
        widgetid: string;
    };

    // ds.LogLine
    type LogLine = {
        linenum: number;
        ts: number;
        msg: string;
        source?: string;
    };

    // rpc.RpcMessage
    type RpcMessage = {
        command?: string;
        reqid?: string;
        resid?: string;
        timeout?: number;
        route?: string;
        authtoken?: string;
        source?: string;
        cont?: boolean;
        cancel?: boolean;
        error?: string;
        datatype?: string;
        data?: any;
    };

    // rpc.RpcOpts
    type RpcOpts = {
        timeout?: number;
        noresponse?: boolean;
        route?: string;
    };

    // rpctypes.SearchRequestData
    type SearchRequestData = {
        widgetid: string;
        searchterm: string;
        offset: number;
        limit: number;
        buffer: number;
        stream: boolean;
    };

    // rpctypes.SearchResultData
    type SearchResultData = {
        widgetid: string;
        filteredcount: number;
        totalcount: number;
        lines: LogLine[];
    };

    // rpctypes.ServerCommandMeta
    type ServerCommandMeta = {
        commandtype: string;
    };

    // rpctypes.StatusUpdateData
    type StatusUpdateData = {
        appname: string;
        status: string;
        numloglines: number;
        numgoroutines: number;
    };

    // rpctypes.StreamUpdateData
    type StreamUpdateData = {
        widgetid: string;
        filteredcount: number;
        totalcount: number;
        lines: LogLine[];
    };

}

export {}
