// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import React from "react";

interface NoWatchesMessageProps {
    hideTitle?: boolean;
}

export const NoWatchesMessage: React.FC<NoWatchesMessageProps> = ({ hideTitle = false }) => {
    return (
        <div className="flex flex-col items-center min-h-full p-6 pt-6 overflow-auto">
            <div className="max-w-4xl mx-auto w-full">
                {!hideTitle && (
                    <>
                        <h2 className="text-2xl md:text-3xl font-bold text-center text-primary">No Watches Found</h2>
                        <p className="text-lg text-center mb-4 text-secondary">
                            Add watches to your Go application to monitor values in real-time.
                        </p>
                    </>
                )}

                <div className="bg-secondary/5 rounded-lg p-6 mb-8">
                    <h3 className="text-xl font-semibold mb-4 text-primary">Available Watch Functions</h3>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                        <div className="space-y-1">
                            <h4 className="font-semibold text-primary">Push Values</h4>
                            <p className="text-sm text-secondary mb-2 pl-4">Push values directly from your code:</p>
                            <div className="space-y-3 pl-4">
                                <div>
                                    <div className="font-mono text-accent">TrackValue(name, val)</div>
                                    <div className="text-sm text-secondary">Push any value</div>
                                </div>
                            </div>
                        </div>

                        <div className="space-y-1">
                            <h4 className="font-semibold text-primary">Poll Values</h4>
                            <p className="text-sm text-secondary mb-2 pl-4">
                                Register funcs to be polled automatically:
                            </p>
                            <div className="space-y-3 pl-4">
                                <div>
                                    <div className="font-mono text-accent">WatchFunc(name, getFn)</div>
                                    <div className="text-sm text-secondary">
                                        Poll any value w/ custom synchronization
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="space-y-1">
                            <h4 className="font-semibold text-primary">Thread-Safe Watching</h4>
                            <p className="text-sm text-secondary mb-2 pl-4">
                                Watch values with automatic synchronization:
                            </p>
                            <div className="space-y-3 pl-4">
                                <div>
                                    <div className="font-mono text-accent">WatchSync(name, lock, val)</div>
                                </div>
                            </div>
                        </div>

                        <div className="space-y-1">
                            <h4 className="font-semibold text-primary">Atomic Values</h4>
                            <p className="text-sm text-secondary mb-2 pl-4">Watch atomic values directly:</p>
                            <div className="space-y-3 pl-4">
                                <div>
                                    <div className="font-mono text-accent">WatchAtomic(name, val)</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="mt-6 text-sm text-secondary italic">
                        Note: Values are polled every second by default. Counter versions of these functions
                        (TrackCounter, WatchCounterFunc, WatchCounterSync, WatchAtomicCounter) are also available for
                        tracking numeric counters.
                    </div>
                </div>

                <div className="bg-secondary/5 p-6 rounded-lg font-mono text-sm overflow-x-auto">
                    <h3 className="text-xl font-semibold mb-4 text-primary font-sans">Example Usage</h3>
                    <pre className="whitespace-pre text-primary">
                        <code>
                            <span className="text-accent">// Track a value</span>
                            <br />
                            outrig.TrackValue("user.profile", user)
                            <br />
                            <br />
                            <span className="text-accent">// Watch a counter that updates automatically</span>
                            <br />
                            counter := atomic.Int64{}
                            <br />
                            outrig.WatchAtomic("requests.count", &counter)
                            <br />
                            <br />
                            <span className="text-accent">// Watch a value with a function</span>
                            <br />
                            outrig.WatchFunc("cache.size", func() int {"{"}
                            <br />
                            {"    "}return len(myCache)
                            <br />
                            {"}"})
                            <br />
                            <br />
                            <span className="text-accent">// Watch a value with mutex protection</span>
                            <br />
                            outrig.WatchSync("app.state", &mu, &appState)
                        </code>
                    </pre>
                </div>
            </div>
        </div>
    );
};
