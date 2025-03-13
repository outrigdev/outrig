// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// generated by cmd/generate/main-generatets.go

import { RpcClient } from "./rpc";

class RpcApiType {
    // command "eventpublish" [call]
    EventPublishCommand(client: RpcClient, data: EventType, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("eventpublish", data, opts);
    }

    // command "eventreadhistory" [call]
    EventReadHistoryCommand(client: RpcClient, data: EventReadHistoryData, opts?: RpcOpts): Promise<EventType[]> {
        return client.rpcCall("eventreadhistory", data, opts);
    }

    // command "eventsub" [call]
    EventSubCommand(client: RpcClient, data: SubscriptionRequest, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("eventsub", data, opts);
    }

    // command "eventunsub" [call]
    EventUnsubCommand(client: RpcClient, data: string, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("eventunsub", data, opts);
    }

    // command "eventunsuball" [call]
    EventUnsubAllCommand(client: RpcClient, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("eventunsuball", null, opts);
    }

    // command "getapprungoroutines" [call]
    GetAppRunGoroutinesCommand(client: RpcClient, data: AppRunRequest, opts?: RpcOpts): Promise<AppRunGoroutinesData> {
        return client.rpcCall("getapprungoroutines", data, opts);
    }

    // command "getapprunlogs" [call]
    GetAppRunLogsCommand(client: RpcClient, data: AppRunRequest, opts?: RpcOpts): Promise<AppRunLogsData> {
        return client.rpcCall("getapprunlogs", data, opts);
    }

    // command "getappruns" [call]
    GetAppRunsCommand(client: RpcClient, data: AppRunUpdatesRequest, opts?: RpcOpts): Promise<AppRunsData> {
        return client.rpcCall("getappruns", data, opts);
    }

    // command "loggetmarkedlines" [call]
    LogGetMarkedLinesCommand(client: RpcClient, data: MarkedLinesRequestData, opts?: RpcOpts): Promise<MarkedLinesResultData> {
        return client.rpcCall("loggetmarkedlines", data, opts);
    }

    // command "logsearchrequest" [call]
    LogSearchRequestCommand(client: RpcClient, data: SearchRequestData, opts?: RpcOpts): Promise<SearchResultData> {
        return client.rpcCall("logsearchrequest", data, opts);
    }

    // command "logstreamupdate" [call]
    LogStreamUpdateCommand(client: RpcClient, data: StreamUpdateData, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("logstreamupdate", data, opts);
    }

    // command "logupdatemarkedlines" [call]
    LogUpdateMarkedLinesCommand(client: RpcClient, data: MarkedLinesData, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("logupdatemarkedlines", data, opts);
    }

    // command "logwidgetadmin" [call]
    LogWidgetAdminCommand(client: RpcClient, data: LogWidgetAdminData, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("logwidgetadmin", data, opts);
    }

    // command "message" [call]
    MessageCommand(client: RpcClient, data: CommandMessageData, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("message", data, opts);
    }

    // command "updatestatus" [call]
    UpdateStatusCommand(client: RpcClient, data: StatusUpdateData, opts?: RpcOpts): Promise<void> {
        return client.rpcCall("updatestatus", data, opts);
    }

}

export const RpcApi = new RpcApiType();
