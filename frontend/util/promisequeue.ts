// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

export interface QueueItem<T> {
    task: () => Promise<T>;
    isStillValid?: () => boolean;
    resolve: (value: T) => void;
    reject: (reason?: any) => void;
}

export class PromiseQueue {
    queue: QueueItem<any>[] = [];
    isRunning = false;

    enqueue<T>(task: () => Promise<T>, isStillValid?: () => boolean): Promise<T> {
        const rtn = new Promise<T>((resolve, reject) => {
            this.queue.push({ task, isStillValid, resolve, reject });
        });
        if (!this.isRunning) {
            this.processQueue();
        }
        return rtn;
    }

    clearQueue(reason: any = new Error("Queue cleared")) {
        // reject all items in the queue
        for (const item of this.queue) {
            item.reject(reason);
        }
        this.queue = [];
    }

    async processQueue() {
        if (this.isRunning || this.queue.length === 0) {
            return;
        }
        this.isRunning = true;
        try {
            while (this.queue.length > 0) {
                const item = this.queue.shift();
                if (!item) {
                    continue;
                }
                if (item.isStillValid != null && !item.isStillValid()) {
                    item.reject(new Error("Task no longer valid"));
                    continue;
                }
                try {
                    const result = await item.task();
                    item.resolve(result);
                } catch (e) {
                    item.reject(e);
                }
            }
        } finally {
            this.isRunning = false;
        }
    }
}
