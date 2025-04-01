import React from "react";
import { Button } from "./ui/button"; // Adjust the path as needed
import { useNavigate } from "react-router-dom";

type JudgmentData = {
  opening_statement: { user: { score: number; reason: string }; bot: { score: number; reason: string } };
  cross_examination: { user: { score: number; reason: string }; bot: { score: number; reason: string } };
  answers: { user: { score: number; reason: string }; bot: { score: number; reason: string } };
  closing: { user: { score: number; reason: string }; bot: { score: number; reason: string } };
  total: { user: number; bot: number };
  verdict: { winner: string; reason: string; congratulations: string; opponent_analysis: string };
};

type JudgmentPopupProps = {
  judgment: JudgmentData;
  userAvatar: string;
  botAvatar: string;
  botName: string;
  userStance: string;
  botStance: string;
  botDesc: string; // Added bot description for profile
  onClose: () => void; // Added onClose prop from DebateRoom
};

const JudgmentPopup: React.FC<JudgmentPopupProps> = ({
  judgment,
  userAvatar,
  botAvatar,
  botName,
  userStance,
  botStance,
  botDesc,
  onClose,
}) => {
  const navigate = useNavigate();
  const userName = "You"; // Could be dynamic if user data is available

  const handleGoHome = () => {
    navigate("/startdebate");
  };

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-60 z-50 p-4">
      <div className="bg-gradient-to-br from-white to-gray-100 p-8 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-y-auto border border-gray-200 transform transition-all duration-300 scale-100 hover:scale-102">
        {/* Top Profile Section */}
        <div className="flex justify-between items-center mb-8">
          {/* User Profile */}
          <div className="flex items-center space-x-4 bg-gray-50 p-4 rounded-lg shadow-sm w-1/2 mr-2">
            <img
              src={userAvatar}
              alt={userName}
              className="w-16 h-16 rounded-full border-2 border-orange-400 object-cover"
            />
            <div>
              <h3 className="text-xl font-bold text-gray-800">{userName}</h3>
              <p className="text-sm text-gray-600">
                Stance: <span className="font-semibold text-orange-500">{userStance}</span>
              </p>
              <p className="text-xs text-gray-500">Debater</p>
            </div>
          </div>
          {/* Bot Profile */}
          <div className="flex items-center space-x-4 bg-gray-50 p-4 rounded-lg shadow-sm w-1/2 ml-2">
            <img
              src={botAvatar}
              alt={botName}
              className="w-16 h-16 rounded-full border-2 border-orange-400 object-cover"
            />
            <div>
              <h3 className="text-xl font-bold text-gray-800">{botName}</h3>
              <p className="text-sm text-gray-600">
                Stance: <span className="font-semibold text-orange-500">{botStance}</span>
              </p>
              <p className="text-xs text-gray-500">{botDesc}</p>
            </div>
          </div>
        </div>

        {/* Phase Sections */}
        <div className="space-y-10">
          {/* Opening Statement */}
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-2xl font-bold text-gray-800 text-center mb-6">Opening Statement</h3>
            <div className="grid grid-cols-2 gap-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{userName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.opening_statement.user.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.opening_statement.user.reason}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{botName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.opening_statement.bot.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.opening_statement.bot.reason}</p>
              </div>
            </div>
          </div>

          {/* Cross Examination */}
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-2xl font-bold text-gray-800 text-center mb-6">Cross Examination</h3>
            <div className="grid grid-cols-2 gap-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{userName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.cross_examination.user.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.cross_examination.user.reason}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{botName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.cross_examination.bot.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.cross_examination.bot.reason}</p>
              </div>
            </div>
          </div>

          {/* Answers */}
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-2xl font-bold text-gray-800 text-center mb-6">Answers to Cross Examination</h3>
            <div className="grid grid-cols-2 gap-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{userName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.answers.user.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.answers.user.reason}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{botName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.answers.bot.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.answers.bot.reason}</p>
              </div>
            </div>
          </div>

          {/* Closing Statement */}
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-2xl font-bold text-gray-800 text-center mb-6">Closing Statement</h3>
            <div className="grid grid-cols-2 gap-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{userName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.closing.user.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.closing.user.reason}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <h4 className="text-lg font-semibold text-gray-700">{botName}</h4>
                <p className="mt-2 text-xl font-bold text-orange-600">{judgment.closing.bot.score}/10</p>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{judgment.closing.bot.reason}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Total Scores */}
        <div className="mt-10 bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-2xl font-bold text-gray-800 text-center mb-6">Total Scores</h3>
          <div className="flex justify-around items-center">
            <div className="text-center">
              <p className="text-4xl font-bold text-orange-600">{judgment.total.user}</p>
              <p className="text-sm text-gray-500">/ 40</p>
              <p className="text-lg font-semibold text-gray-700">{userName}</p>
            </div>
            <div className="text-center">
              <p className="text-4xl font-bold text-orange-600">{judgment.total.bot}</p>
              <p className="text-sm text-gray-500">/ 40</p>
              <p className="text-lg font-semibold text-gray-700">{botName}</p>
            </div>
          </div>
        </div>

        {/* Verdict */}
        <div className="mt-10 bg-gradient-to-r from-orange-500 to-orange-600 p-6 rounded-lg shadow-md text-white text-center">
          <h3 className="text-2xl font-bold">Verdict</h3>
          <p className="mt-4 text-3xl font-bold">{judgment.verdict.winner} Wins!</p>
          <p className="mt-3 text-lg">{judgment.verdict.congratulations}</p>
          <p className="mt-2 text-md leading-relaxed">{judgment.verdict.opponent_analysis}</p>
        </div>

        {/* Go Home Button */}
        <div className="text-center mt-8">
          <Button
            onClick={handleGoHome}
            className="bg-orange-500 hover:bg-orange-600 text-white px-6 py-3 rounded-full text-lg font-semibold transition-all duration-200 transform hover:scale-105 mr-4"
          >
            Back to Home
          </Button>
          <Button
            onClick={onClose}
            className="bg-gray-500 hover:bg-gray-600 text-white px-6 py-3 rounded-full text-lg font-semibold transition-all duration-200 transform hover:scale-105"
          >
            Close
          </Button>
        </div>
      </div>
    </div>
  );
};

export default JudgmentPopup;