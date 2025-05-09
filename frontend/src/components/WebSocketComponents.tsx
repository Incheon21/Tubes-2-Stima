// src/components/WebSocketComponent.tsx
import React, { useEffect, useState } from "react";

const WebSocketComponent: React.FC = () => {
    const [message, setMessage] = useState<string>("");

    useEffect(() => {
        const socket = new WebSocket("ws://localhost:8080/ws");

        // Listen for messages
        socket.onmessage = (event) => {
            console.log("Received message:", event.data);
            setMessage(event.data);
        };

        // Handle connection open
        socket.onopen = () => {
            console.log("WebSocket connection established");
            socket.send("Hello from client!"); // Send a message to the server
        };

        // Handle errors
        socket.onerror = (error) => {
            console.error("WebSocket error:", error);
        };

        // Cleanup the WebSocket connection on component unmount
        return () => {
            socket.close();
        };
    }, []);

    return (
        <div>
            <h1>WebSocket Message:</h1>
            <p>{message}</p>
        </div>
    );
};

export default WebSocketComponent;
