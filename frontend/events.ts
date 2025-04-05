import mitt from "mitt";

// Define event types
export type Events = {
    logstreamupdate: StreamUpdateData;
    modalclose: void; // Event emitted when a modal is closed
    // Add more events here as needed
};

// Create and export the event emitter
export const emitter = mitt<Events>();
