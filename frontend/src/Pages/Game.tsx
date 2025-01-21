import React, { useState, useEffect, useRef } from "react";
import { useParams } from "react-router-dom";
import PlayerCard from "../components/PlayerCard";
import UserCamera from "../components/UserCamera";
import Chatbox from "../components/Chatbox";

const Game: React.FC = () => {
  const { userId } = useParams<{ userId: string }>();
  const [cameraOn, setCameraOn] = useState<boolean>(true);
  const [micOn, setMicOn] = useState<boolean>(true);
  const [loading, setLoading] = useState<boolean>(true);
  const [gameEnded, setGameEnded] = useState<boolean>(false);
  const [isTurn, setIsTurn] = useState<boolean>(false);
  const [turnDuration, setTurnDuration] = useState<number>(0);
  const websocketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const wsURL = `${import.meta.env.VITE_BASE_URL}/ws?userId=${userId}`;
    const ws = new WebSocket(wsURL);
    ws.binaryType = "arraybuffer";
    websocketRef.current = ws;
    

    ws.onopen = () => {
      console.log("WebSocket connection established");
    };

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      console.log(message);
      switch (message.type) {
        case "DEBATE_START": {
          setLoading(false);
          break;
        }
        case "DEBATE_END": {
          setGameEnded(true);
          break;
        }
        case "TURN_START": {
          const { currentTurn, duration } = JSON.parse(message.content);
          setIsTurn(currentTurn === userId);
          setTurnDuration(duration);
          console.log(currentTurn === userId)
          break;
        }
        default:
          console.warn("Unhandled message type:", message.type);
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    ws.onclose = () => {
      console.log("WebSocket connection closed");
    };
  }, []);

  const renderContent = () => {
    if (loading) {
      return <div className="w-screen h-screen flex justify-center items-center">Loading...</div>;
    }

    if (gameEnded) {
      return <div className="w-screen h-screen flex justify-center items-center">Game Ended</div>;
    }


    return (
      <div className="w-screen h-screen flex justify-center items-center">
        <div className="flex flex-col w-full md:w-1/2 h-4/5 gap-y-2 border rounded">
          <PlayerCard 
            isUser={false} 
            isTurn={!isTurn} 
            turnDuration={turnDuration} 
          />
          <div className="flex flex-col md:flex-row h-full">
            <div className="flex-1 h-full">
              <UserCamera
                cameraOn={true}
                micOn={true}
                sendData={false}
                websocket={websocketRef.current}
              />
            </div>
            <div className="flex-1 h-full">
              <UserCamera
                cameraOn={cameraOn}
                micOn={micOn}
                sendData={isTurn}
                websocket={websocketRef.current}
              />
            </div>
          </div>
          <PlayerCard
            isUser={true}
            cameraOn={cameraOn}
            micOn={micOn}
            setCameraOn={setCameraOn}
            setMicOn={setMicOn}
            isTurn={isTurn}
            turnDuration={turnDuration}
          />
        </div>
        <div className="w-full md:w-1/4 h-4/5 flex flex-col border rounded">
          <Chatbox />
        </div>
      </div>
    );
  };

  return renderContent();
};

export default Game;
