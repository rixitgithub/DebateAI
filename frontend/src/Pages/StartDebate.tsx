import React, { useContext, useState } from "react";
import { useNavigate } from "react-router-dom";
import { RiRobot2Fill } from "react-icons/ri";
import { FaHandshakeSimpleSlash } from "react-icons/fa6";

import DebateCover from "../assets/DebateCover4.svg";
import { Button } from "../components/ui/button";
import { AuthContext } from "../context/authContext";
import DebatePopup from "@/components/DebatePopup";

const StartDebate = () => {
  const navigate = useNavigate();
  const authContext = useContext(AuthContext);
  const [showPopup, setShowPopup] = useState(false);

  const handlePlayDebateClick = () => {
    if (authContext?.isAuthenticated) {
      setShowPopup(true);
    } else {
      navigate("/auth", { state: { isSignUp: false } });
    }
  };

  const handlePlayBotClick = () => {
    if (authContext?.isAuthenticated) {
      navigate("/bot-selection");
    } else {
      navigate("/auth", { state: { isSignUp: false } });
    }
  };

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-center flex-1">
        <div className="flex flex-wrap items-center justify-center w-full px-2 md:px-16">
          <div className="w-full md:w-2/3 p-4 md:p-16">
            <img src={DebateCover} alt="Debate Cover" className="w-full object-cover" />
          </div>
          <div className="flex w-full md:w-1/3 flex-col items-center justify-center space-y-4 p-4">
            <h3 className="text-xl md:text-4xl font-bold text-center">
              Play Debate Online on the <span className="text-primary">#1</span> Site!
            </h3>
            <div className="flex flex-col w-full">
              <Button
                className="my-2 h-auto rounded text-xl flex items-center justify-start"
                onClick={handlePlayDebateClick}
              >
                <FaHandshakeSimpleSlash className="text-4xl" />
                <div className="flex flex-col items-start ml-4">
                  <span className="font-bold">Play Online</span>
                  <span className="text-sm text-primary-foreground font-thin">
                    Play with someone at your level
                  </span>
                </div>
              </Button>
              <Button
                className="my-2 h-auto rounded text-xl flex items-center justify-start"
                variant="outline"
                onClick={handlePlayBotClick}
              >
                <RiRobot2Fill className="text-4xl" />
                <div className="flex flex-col items-start ml-4">
                  <span className="font-bold">Practice with Bot</span>
                  <span className="text-sm text-muted-foreground font-thin">
                    Improve your skills with AI guidance
                  </span>
                </div>
              </Button>
            </div>
          </div>
        </div>
      </div>
      {showPopup && <DebatePopup onClose={() => setShowPopup(false)} />}
    </div>
  );
};

export default StartDebate;
