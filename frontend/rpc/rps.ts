// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { DefaultRpcClient } from "@/init";
import { isBlank } from "../util/util";
import { RpcApi } from "./rpcclientapi";

type EventSubjectType = {
    handler: (event: EventType) => void;
    scope?: string;
};

type EventSubjectContainer = EventSubjectType & {
    id: string;
};

type EventSubscription = EventSubjectType & {
    eventType: string;
};

type EventUnsubscribe = {
    id: string;
    eventType: string;
};

// key is "eventType" or "eventType|oref"
const eventSubjects = new Map<string, EventSubjectContainer[]>();

function eventReconnectHandler() {
    for (const eventType of eventSubjects.keys()) {
        updateEventSub(eventType);
    }
}

function updateEventSub(eventType: string) {
    let subjects = eventSubjects.get(eventType);
    if (subjects == null) {
        RpcApi.EventUnsubCommand(DefaultRpcClient, eventType, { noresponse: true });
        return;
    }
    let subreq: SubscriptionRequest = { event: eventType, scopes: [], allscopes: false };
    for (const scont of subjects) {
        if (isBlank(scont.scope)) {
            subreq.allscopes = true;
            subreq.scopes = [];
            break;
        }
        subreq.scopes.push(scont.scope);
    }
    RpcApi.EventSubCommand(DefaultRpcClient, subreq, { noresponse: true });
}

function eventSubscribe(...subscriptions: EventSubscription[]): () => void {
    const unsubs: EventUnsubscribe[] = [];
    const eventTypeSet = new Set<string>();
    for (const subscription of subscriptions) {
        if (subscription.handler == null) {
            return;
        }
        const id: string = crypto.randomUUID();
        let subjects = eventSubjects.get(subscription.eventType);
        if (subjects == null) {
            subjects = [];
            eventSubjects.set(subscription.eventType, subjects);
        }
        const subcont: EventSubjectContainer = { id, handler: subscription.handler, scope: subscription.scope };
        subjects.push(subcont);
        unsubs.push({ id, eventType: subscription.eventType });
        eventTypeSet.add(subscription.eventType);
    }
    for (const eventType of eventTypeSet) {
        updateEventSub(eventType);
    }
    return () => eventUnsubscribe(...unsubs);
}

function eventUnsubscribe(...unsubscribes: EventUnsubscribe[]) {
    const eventTypeSet = new Set<string>();
    for (const unsubscribe of unsubscribes) {
        let subjects = eventSubjects.get(unsubscribe.eventType);
        if (subjects == null) {
            return;
        }
        const idx = subjects.findIndex((s) => s.id === unsubscribe.id);
        if (idx === -1) {
            return;
        }
        subjects.splice(idx, 1);
        if (subjects.length === 0) {
            eventSubjects.delete(unsubscribe.eventType);
        }
        eventTypeSet.add(unsubscribe.eventType);
    }

    for (const eventType of eventTypeSet) {
        updateEventSub(eventType);
    }
}

function handleRecvEvent(event: EventType) {
    const subjects = eventSubjects.get(event.event);
    if (subjects == null) {
        return;
    }
    for (const scont of subjects) {
        if (isBlank(scont.scope)) {
            scont.handler(event);
            continue;
        }
        if (event.scopes == null) {
            continue;
        }
        if (event.scopes.includes(scont.scope)) {
            scont.handler(event);
        }
    }
}

export { eventReconnectHandler, eventSubscribe, eventUnsubscribe, handleRecvEvent };
