// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useRef, useState } from "react";

/**
 * A generic hook for initializing and managing Outrig model classes.
 * 
 * This hook handles the common pattern of:
 * 1. Creating a model instance with an appRunId
 * 2. Storing it in a ref
 * 3. Triggering a re-render when the model is initialized
 * 4. Cleaning up the model when the component unmounts
 * 
 * @param ModelClass - The model class constructor
 * @param appRunId - The ID of the app run
 * @returns The initialized model instance or null if not yet initialized
 */
export function useOutrigModel<T extends { dispose: () => void }>(
  ModelClass: new (appRunId: string) => T,
  appRunId: string
): T | null {
  const modelRef = useRef<T | null>(null);
  const [, setForceUpdate] = useState({});

  useEffect(() => {
    if (!modelRef.current) {
      // Initialize the model with the appRunId
      modelRef.current = new ModelClass(appRunId);
      
      // Force a re-render to make the model available to the component
      setForceUpdate({});
    }
    
    // Cleanup function to dispose the model when the component unmounts
    // or when the appRunId changes
    return () => {
      if (modelRef.current) {
        modelRef.current.dispose();
        modelRef.current = null;
      }
    };
  }, [ModelClass, appRunId]);

  return modelRef.current;
}
