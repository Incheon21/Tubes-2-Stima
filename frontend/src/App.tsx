// src/App.tsx
import React from "react";
import WebSocketComponent from "./components/WebSocketComponents.tsx";
import ServerClock from "./components/ServerClock.tsx";

const App: React.FC = () => {
    return (
        <div>
            <WebSocketComponent />
            <ServerClock />
        </div>
    );
};

export default App;
