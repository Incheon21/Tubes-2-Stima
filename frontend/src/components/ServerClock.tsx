// src/components/ServerClock.tsx
import { useEffect, useState } from "react";

const ServerClock = () => {
  const [time, setTime] = useState<string>("Connecting...");

  useEffect(() => {
    const ws = new WebSocket("ws://localhost:8080/ws"); // adjust if hosted elsewhere

    ws.onopen = () => {
      console.log("WebSocket connected");
    };

    ws.onmessage = (event) => {
      setTime(event.data);
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    ws.onclose = () => {
      setTime("Connection closed");
    };

    return () => {
      ws.close();
    };
  }, []);

  return (
    <div>
      <h2>Server Time</h2>
      <p style={{ fontSize: "2em" }}>{time}</p>
    </div>
  );
};

export default ServerClock;
