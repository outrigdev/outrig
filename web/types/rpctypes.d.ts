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
        numactivewatches: number;
        numtotalwatches: number;
        lastmodtime: number;
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
        since?: number;
    };

    // rpctypes.AppRunUpdatesRequest
    type AppRunUpdatesRequest = {
        since: number;
    };

    // rpctypes.AppRunWatchesData
    type AppRunWatchesData = {
        apprunid: string;
        appname: string;
        watches: Watch[];
    };

    // rpctypes.AppRunsData
    type AppRunsData = {
        appruns: AppRunInfo[];
    };

    // rpctypes.BrowserTabUrlData
    type BrowserTabUrlData = {
        url: string;
        apprunid?: string;
    };

    // rpctypes.CommandMessageData
    type CommandMessageData = {
        message: string;
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

    // rpctypes.LogWidgetAdminData
    type LogWidgetAdminData = {
        widgetid: string;
        drop?: boolean;
        keepalive?: boolean;
    };

    // rpctypes.MarkedLinesData
    type MarkedLinesData = {
        widgetid: string;
        markedlines: {[key: string]: boolean};
        clear?: boolean;
    };

    // rpctypes.MarkedLinesRequestData
    type MarkedLinesRequestData = {
        widgetid: string;
    };

    // rpctypes.MarkedLinesResultData
    type MarkedLinesResultData = {
        lines: LogLine[];
    };

    // rpctypes.PageData
    type PageData = {
        pagenum: number;
        lines: LogLine[];
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
        apprunid: string;
        searchterm: string;
        searchtype?: string;
        pagesize: number;
        requestpages: number[];
        stream: boolean;
    };

    // rpctypes.SearchResultData
    type SearchResultData = {
        filteredcount: number;
        totalcount: number;
        pages: PageData[];
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

    // ds.Watch
    type Watch = {
        ts: number;
        name: string;
        value?: string;
        type: string;
        error?: string;
        addr?: string[];
        cap?: number;
        len?: number;
        waittime?: number;
    };

}

export {}
