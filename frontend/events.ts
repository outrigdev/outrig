import mitt from 'mitt';

// Define event types
export type Events = {
  logstreamupdate: StreamUpdateData;
  // Add more events here as needed
};

// Create and export the event emitter
export const emitter = mitt<Events>();
