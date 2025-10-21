import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import './EventStream.css';

const EventStream = () => {
  const [events, setEvents] = useState([]);
  const { lastEvent } = useWebSocket('ws://localhost:8080/api/v1/events/stream');

  useEffect(() => {
    if (lastEvent) {
      setEvents((prevEvents) => {
        const newEvents = [lastEvent, ...prevEvents];
        return newEvents.slice(0, 100); // Keep only last 100 events
      });
    }
  }, [lastEvent]);

  const formatTimestamp = (timestamp) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  const getEventClass = (eventType) => {
    switch (eventType) {
      case 'network':
        return 'event-network';
      case 'process':
        return 'event-process';
      default:
        return 'event-other';
    }
  };

  return (
    <div className="event-stream">
      <h2>Live Event Stream</h2>
      <div className="event-list">
        {events.map((event, index) => (
          <div key={index} className={`event-item ${getEventClass(event.type)}`}>
            <div className="event-header">
              <span className="event-type">{event.type}</span>
              <span className="event-hostname">{event.hostname}</span>
              <span className="event-time">{formatTimestamp(event.timestamp)}</span>
            </div>
            <div className="event-body">
              <pre>{JSON.stringify(event.data, null, 2)}</pre>
            </div>
          </div>
        ))}
        {events.length === 0 && (
          <div className="no-data">Waiting for events...</div>
        )}
      </div>
    </div>
  );
};

export default EventStream;
