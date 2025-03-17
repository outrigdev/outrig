// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// generated by cmd/generate/main-generatets.go

declare global {

    // rpctypes.AppRunGoRoutinesData
    type AppRunGoRoutinesData = {
        apprunid: string;
        appname: string;
        goroutines: ParsedGoRoutine[];
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
        buildinfo?: BuildInfoData;
        modulename?: string;
        executable?: string;
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

    // rpctypes.AppRunRuntimeStatsData
    type AppRunRuntimeStatsData = {
        apprunid: string;
        appname: string;
        stats: RuntimeStatData[];
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

    // rpctypes.BuildInfoData
    type BuildInfoData = {
        goversion: string;
        path: string;
        version?: string;
        settings?: {[key: string]: string};
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

    // ds.MemoryStatsInfo
    type MemoryStatsInfo = {
        alloc: number;
        totalalloc: number;
        sys: number;
        heapalloc: number;
        heapsys: number;
        heapidle: number;
        heapinuse: number;
        stackinuse: number;
        stacksys: number;
        mspaninuse: number;
        mspansys: number;
        mcacheinuse: number;
        mcachesys: number;
        gcsys: number;
        othersys: number;
        nextgc: number;
        lastgc: number;
        pausetotalns: number;
        numgc: number;
    };

    // rpctypes.PageData
    type PageData = {
        pagenum: number;
        lines: LogLine[];
    };

    // rpctypes.ParsedGoRoutine
    type ParsedGoRoutine = {
        goid: number;
        rawstacktrace: string;
        rawstate: string;
        primarystate: string;
        statedurationms?: number;
        extrastates?: string[];
        parsedframes?: StackFrame[];
        createdbygoid?: number;
        createdbyframe?: StackFrame;
        parsed: boolean;
        parseerror?: string;
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

    // rpctypes.RuntimeStatData
    type RuntimeStatData = {
        ts: number;
        cpuusage: number;
        goroutinecount: number;
        gomaxprocs: number;
        numcpu: number;
        goos: string;
        goarch: string;
        goversion: string;
        pid: number;
        cwd: string;
        memstats: MemoryStatsInfo;
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

    // rpctypes.StackFrame
    type StackFrame = {
        package: string;
        funcname: string;
        funcargs?: string;
        filepath: string;
        linenumber: number;
        pcoffset?: string;
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
