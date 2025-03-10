// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// generated by cmd/generate/main-generatets.go

declare global {

    // rpctypes.AppRunGoroutinesData
    type AppRunGoroutinesData = {
        apprunid: string;
        appname: string;
        goroutines: GoroutineData[];
    };

    // rpctypes.AppRunInfo
    type AppRunInfo = {
        apprunid: string;
        appname: string;
        starttime: number;
        isrunning: boolean;
        status: string;
        numlogs: number;
        numactivegoroutines: number;
        numtotalgoroutines: number;
    };

    // rpctypes.AppRunLogsData
    type AppRunLogsData = {
        apprunid: string;
        appname: string;
        logs: LogLine[];
    };

    // rpctypes.AppRunRequest
    type AppRunRequest = {
        apprunid: string;
    };

    // rpctypes.AppRunsData
    type AppRunsData = {
        appruns: AppRunInfo[];
    };

    // rpctypes.CommandMessageData
    type CommandMessageData = {
        message: string;
    };

    // rpctypes.DropRequestData
    type DropRequestData = {
        widgetid: string;
    };

    // rpctypes.EventCommonFields
    type EventCommonFields = {
        scopes?: string[];
        sender?: string;
        persist?: number;
    };

    // rpctypes.EventReadHistoryData
    type EventReadHistoryData = {
        event: string;
        scope: string;
        maxitems: number;
    };

    // EventType union (rpctypes.EventToTypeMap)
    type EventType = 
        | (EventCommonFields & { event: "route:down"; data?: null })
        | (EventCommonFields & { event: "route:up"; data?: null })
        | (EventCommonFields & { event: "app:statusupdate"; data: StatusUpdateData })
    ;

    // rpctypes.GoroutineData
    type GoroutineData = {
        goid: number;
        state: string;
        stacktrace: string;
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
        appid: string;
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

    // rpctypes.SubscriptionRequest
    type SubscriptionRequest = {
        event: string;
        scopes?: string[];
        allscopes?: boolean;
    };

}

export {}
