import React, { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { Button } from "../components/ui/button";

import JudgmentPopup from "@/components/JudgementPopup";
import SpeechTranscripts from "@/components/SpeechTranscripts";
import { useUser } from "@/hooks/useUser";
import { getAuthToken } from "@/utils/auth";

// Define debate phases as an enum
enum DebatePhase {
  Setup = "setup",
  OpeningFor = "openingFor",
  OpeningAgainst = "openingAgainst",
  CrossForQuestion = "crossForQuestion",
  CrossAgainstAnswer = "crossAgainstAnswer",
  CrossAgainstQuestion = "crossAgainstQuestion",
  CrossForAnswer = "crossForAnswer",
  ClosingFor = "closingFor",
  ClosingAgainst = "closingAgainst",
  Finished = "finished",
}

// Define debate roles
type DebateRole = "for" | "against";

type JudgmentData = {
  opening_statement: {
    for: { score: number; reason: string };
    against: { score: number; reason: string };
  };
  cross_examination_questions: {
    for: { score: number; reason: string };
    against: { score: number; reason: string };
  };
  cross_examination_answers: {
    for: { score: number; reason: string };
    against: { score: number; reason: string };
  };
  closing: {
    for: { score: number; reason: string };
    against: { score: number; reason: string };
  };
  total: { for: number; against: number };
  verdict: {
    winner: string;
    reason: string;
    congratulations: string;
    opponent_analysis: string;
  };
};

// Define user details interface
interface UserDetails {
  id: string;
  username: string;
  elo: number;
  avatarUrl?: string;
  displayName?: string;
}

// Define WebSocket message structure
interface WSMessage {
  type: string;
  topic?: string;
  role?: DebateRole;
  ready?: boolean;
  phase?: DebatePhase;
  offer?: RTCSessionDescriptionInit;
  answer?: RTCSessionDescriptionInit;
  candidate?: RTCIceCandidateInit;
  message?: string;
  userDetails?: UserDetails;
  roomParticipants?: UserDetails[];
  // Enhanced chat fields
  userId?: string;
  username?: string;
  timestamp?: number;
  mode?: "type" | "speak";
  isTyping?: boolean;
  isSpeaking?: boolean;
  partialText?: string;
  // Automatic muting fields
  isMuted?: boolean;
  currentTurn?: string;
  speechText?: string;
}

// Define phase durations in seconds
const phaseDurations: { [key in DebatePhase]?: number } = {
  [DebatePhase.OpeningFor]: 60,
  [DebatePhase.OpeningAgainst]: 60,
  [DebatePhase.CrossForQuestion]: 30,
  [DebatePhase.CrossAgainstAnswer]: 30,
  [DebatePhase.CrossAgainstQuestion]: 30,
  [DebatePhase.CrossForAnswer]: 30,
  [DebatePhase.ClosingFor]: 45,
  [DebatePhase.ClosingAgainst]: 45,
};

// Function to extract JSON from response
const extractJSON = (response: string): string => {
  const fenceRegex = /```(?:json)?\s*([\s\S]*?)\s*```/;
  const match = fenceRegex.exec(response);
  if (match && match[1]) return match[1].trim();
  return response;
};

const OnlineDebateRoom: React.FC = () => {
  const { roomId } = useParams<{ roomId: string }>();
  const { user: currentUser } = useUser();

  // User state management
  const [localUser, setLocalUser] = useState<UserDetails | null>(null);
  const [opponentUser, setOpponentUser] = useState<UserDetails | null>(null);
  const [roomParticipants, setRoomParticipants] = useState<UserDetails[]>([]);

  // Refs for WebSocket, PeerConnection, and media elements
  const wsRef = useRef<WebSocket | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animationRef = useRef<number | null>(null);
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  // State for debate setup and signaling
  const [topic, setTopic] = useState("");
  const [localRole, setLocalRole] = useState<DebateRole | null>(null);
  const [peerRole, setPeerRole] = useState<DebateRole | null>(null);
  const [localReady, setLocalReady] = useState(false);
  const [peerReady, setPeerReady] = useState(false);
  const [debatePhase, setDebatePhase] = useState<DebatePhase>(
    DebatePhase.Setup
  );

  // State for media streams
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [mediaError, setMediaError] = useState<string | null>(null);

  // State for automatic muting
  const [isAutoMuted, setIsAutoMuted] = useState(false);
  const [speechTranscripts, setSpeechTranscripts] = useState<{
    [key: string]: string;
  }>({});

  // Timer state
  const [timer, setTimer] = useState<number>(0);

  // Audio recording state
  const [isRecording, setIsRecording] = useState(false);
  const [audioLevel, setAudioLevel] = useState(0);
  const [mediaRecorder, setMediaRecorder] = useState<MediaRecorder | null>(
    null
  );
  const [audioStream, setAudioStream] = useState<MediaStream | null>(null);
  const [audioError, setAudioError] = useState<string | null>(null);

  // Speech recognition state
  const [isListening, setIsListening] = useState(false);
  const [currentTranscript, setCurrentTranscript] = useState("");
  const [speechError, setSpeechError] = useState<string | null>(null);
  const [debugMode, setDebugMode] = useState(false);
  const recognitionRef = useRef<SpeechRecognition | null>(null);

  // Popup and countdown state
  const [showSetupPopup, setShowSetupPopup] = useState(true);
  const [countdown, setCountdown] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Predefined debate topics
  const predefinedTopics = [
    "Should social media platforms be regulated more strictly?",
    "Is remote work better than office work?",
    "Should college education be free for everyone?",
    "Are video games beneficial for children?",
    "Should there be a universal basic income?",
    "Is artificial intelligence a threat to humanity?",
    "Should the voting age be lowered to 16?",
    "Are electric vehicles better than traditional cars?",
    "Should junk food be banned in schools?",
    "Is online learning as effective as traditional education?",
  ];

  // Judgment states
  const [popup, setPopup] = useState<{
    show: boolean;
    message: string;
    isJudging?: boolean;
  }>({ show: false, message: "" });
  const [judgmentData, setJudgmentData] = useState<JudgmentData | null>(null);
  const [showJudgment, setShowJudgment] = useState(false);

  // Ordered list of debate phases
  const phaseOrder: DebatePhase[] = [
    DebatePhase.OpeningFor,
    DebatePhase.OpeningAgainst,
    DebatePhase.CrossForQuestion,
    DebatePhase.CrossAgainstAnswer,
    DebatePhase.CrossAgainstQuestion,
    DebatePhase.CrossForAnswer,
    DebatePhase.ClosingFor,
    DebatePhase.ClosingAgainst,
    DebatePhase.Finished,
  ];

  // Determine if it's the local user's turn to speak
  const isMyTurn =
    localRole ===
    (debatePhase.includes("For")
      ? "for"
      : debatePhase.includes("Against")
      ? "against"
      : null);

  // Function to send transcripts to backend
  const sendTranscriptsToBackend = async (
    roomId: string,
    role: DebateRole,
    transcripts: { [key in DebatePhase]?: string }
  ) => {
    const token = getAuthToken();

    try {
      const response = await fetch(`http://localhost:1313/submit-transcripts`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          roomId,
          role,
          transcripts,
        }),
      });
      if (!response.ok) {
        throw new Error(
          `Failed to send transcripts: ${response.status} ${response.statusText}`
        );
      }
      const result = await response.json();
      console.log(`Response from backend for ${role}:`, result);

      if (result.message === "Waiting for opponent submission") {
        // Poll for the result periodically until judgment is available
        const pollResult = async () => {
          const pollInterval = setInterval(async () => {
            const pollResponse = await fetch(
              `http://localhost:1313/submit-transcripts`,
              {
                method: "POST",
                headers: {
                  "Content-Type": "application/json",
                  Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({ roomId, role, transcripts: {} }), // Empty transcripts to just check result
              }
            );
            const pollData = await pollResponse.json();
            if (
              pollData.message === "Debate judged" ||
              pollData.message === "Debate already judged"
            ) {
              clearInterval(pollInterval);
              const jsonString = extractJSON(pollData.result);
              const judgment: JudgmentData = JSON.parse(jsonString);
              setJudgmentData(judgment);
              setPopup({ show: false, message: "" });
              setShowJudgment(true);
            }
          }, 2000); // Poll every 2 seconds
        };
        pollResult();
        return null; // Return null to indicate waiting
      } else if (
        result.message === "Debate judged" ||
        result.message === "Debate already judged"
      ) {
        const jsonString = extractJSON(result.result);
        const judgment: JudgmentData = JSON.parse(jsonString);
        return judgment;
      }
    } catch (error) {
      console.error(`Error submitting transcripts for ${role}:`, error);
      throw error;
    }
  };

  // Log message history, collect transcripts, and send to backend
  const logMessageHistory = async () => {
    if (!localRole) {
      console.log("Cannot log message history: localRole is not defined yet.");
      setPopup({
        show: true,
        message: "Please select a role before the debate ends.",
        isJudging: false,
      });
      return;
    }

    console.log(`logMessageHistory called for role: ${localRole}`);
    console.log("Debate Message History:");
    const debateTranscripts: { [key in DebatePhase]?: string } = {};

    const phasesForRole =
      localRole === "for"
        ? [
            DebatePhase.OpeningFor,
            DebatePhase.CrossForQuestion,
            DebatePhase.CrossForAnswer,
            DebatePhase.ClosingFor,
          ]
        : [
            DebatePhase.OpeningAgainst,
            DebatePhase.CrossAgainstAnswer,
            DebatePhase.CrossAgainstQuestion,
            DebatePhase.ClosingAgainst,
          ];

    phasesForRole.forEach((phase) => {
      const transcript =
        localStorage.getItem(`${roomId}_${phase}_${localRole}`) ||
        "No response";
      debateTranscripts[phase] = transcript;
    });
    console.log(`Collected transcripts for ${localRole}:`, debateTranscripts);

    setPopup({
      show: true,
      message: "Submitting transcripts and awaiting judgment...",
      isJudging: true,
    });

    if (roomId && localRole) {
      try {
        console.log(`Sending transcripts to backend for ${localRole}`);
        const judgment = await sendTranscriptsToBackend(
          roomId,
          localRole,
          debateTranscripts
        );
        if (judgment) {
          setJudgmentData(judgment);
          setPopup({ show: false, message: "" });
          setShowJudgment(true);
        } // If null, polling is already handling the wait
      } catch (error) {
        console.error(
          `Failed to send transcripts to backend for ${localRole}:`,
          error
        );
        setPopup({
          show: false,
          message: "Error occurred while judging. Please try again.",
        });
      }
    } else {
      console.log(
        `Cannot send transcripts. roomId: ${roomId}, localRole: ${localRole}`
      );
      setPopup({ show: false, message: "" });
    }
  };

  // Set timer based on phase duration
  useEffect(() => {
    if (phaseDurations[debatePhase]) {
      setTimer(phaseDurations[debatePhase]!);
    } else {
      setTimer(0);
    }
  }, [debatePhase]);

  // Timer countdown and phase transition
  useEffect(() => {
    if (timer > 0 && debatePhase !== DebatePhase.Finished) {
      timerRef.current = setInterval(() => {
        setTimer((prev) => {
          if (prev <= 1) {
            clearInterval(timerRef.current!);
            if (isMyTurn && localRole) {
              // Save any existing transcript for this phase
              const existingTranscript =
                speechTranscripts[debatePhase] || "No response";
              localStorage.setItem(
                `${roomId}_${debatePhase}_${localRole}`,
                existingTranscript
              );
              console.log(
                `Timer expired for ${localRole} in ${debatePhase}. Transcript saved:`,
                existingTranscript
              );
            }
            handlePhaseDone();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [timer, debatePhase, isMyTurn, speechTranscripts, localRole, roomId]);

  // Function to create a room if it doesn't exist
  const createRoomIfNeeded = async () => {
    if (!roomId || !currentUser) return;

    try {
      const token = getAuthToken();
      const response = await fetch(`http://localhost:1313/rooms`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          type: "public",
          roomId: roomId,
        }),
      });

      if (response.ok) {
        console.log("Room created successfully");
        return true;
      }
    } catch (error) {
      console.error("Failed to create room:", error);
    }
    return false;
  };

  // Function to fetch room participants
  const fetchRoomParticipants = async (retryCount = 0) => {
    if (!roomId) return;

    setIsLoading(true);
    try {
      const token = getAuthToken();
      const response = await fetch(
        `http://localhost:1313/rooms/${roomId}/participants`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (response.ok) {
        const participants = await response.json();
        console.log("Fetched participants:", participants);
        console.log("Current user:", currentUser);
        setRoomParticipants(participants);

        // Set local and opponent user details
        if (currentUser && participants.length === 2) {
          console.log("Current user ID:", currentUser.id);
          console.log("All participants:", participants);

          const localParticipant = participants.find(
            (p: UserDetails) => p.id === currentUser.id
          );
          const opponentParticipant = participants.find(
            (p: UserDetails) => p.id !== currentUser.id
          );

          console.log("Local participant:", localParticipant);
          console.log("Opponent participant:", opponentParticipant);

          if (localParticipant) {
            const localUserData = {
              ...localParticipant,
              avatarUrl: currentUser.avatarUrl || localParticipant.avatarUrl,
              displayName:
                currentUser.displayName || localParticipant.displayName,
            };
            console.log("Setting local user:", localUserData);
            setLocalUser(localUserData);
          } else {
            console.log("Local participant not found!");
          }

          if (opponentParticipant) {
            console.log("Setting opponent user:", opponentParticipant);
            setOpponentUser(opponentParticipant);
          } else {
            console.log("Opponent participant not found!");
          }
        } else {
          console.log("Not enough participants or no current user:", {
            currentUser: !!currentUser,
            participantsLength: participants.length,
          });
        }
      } else {
        console.log(
          "API response not ok:",
          response.status,
          response.statusText
        );

        // If room not found (404), it might still be being created
        if (response.status === 404 && retryCount < 5) {
          console.log(
            `Room not found, might still be creating. Retry ${
              retryCount + 1
            }/5 in 2 seconds...`
          );

          // Try to create the room if it's the first retry
          if (retryCount === 0) {
            await createRoomIfNeeded();
          }

          setTimeout(() => {
            fetchRoomParticipants(retryCount + 1);
          }, 2000);
          return;
        }

        // Fallback: use current user as local user and create a placeholder opponent
        if (currentUser) {
          const fallbackLocalUser = {
            id: currentUser.id || "",
            username: currentUser.displayName || "You",
            elo: currentUser.rating || 1500,
            avatarUrl:
              currentUser.avatarUrl ||
              "https://avatar.iran.liara.run/public/40",
            displayName: currentUser.displayName || "You",
          };
          console.log("Setting fallback local user:", fallbackLocalUser);
          setLocalUser(fallbackLocalUser);

          const fallbackOpponentUser = {
            id: "opponent",
            username: "Opponent",
            elo: 1500,
            avatarUrl: "https://avatar.iran.liara.run/public/31",
            displayName: "Opponent",
          };
          console.log("Setting fallback opponent user:", fallbackOpponentUser);
          setOpponentUser(fallbackOpponentUser);
        }
      }
    } catch (error) {
      console.error("Failed to fetch room participants:", error);
      // Fallback: use current user as local user and create a placeholder opponent
      if (currentUser) {
        const errorFallbackLocalUser = {
          id: currentUser.id || "",
          username: currentUser.displayName || "You",
          elo: currentUser.rating || 1500,
          avatarUrl: currentUser.avatarUrl,
          displayName: currentUser.displayName,
        };
        console.log(
          "Setting error fallback local user:",
          errorFallbackLocalUser
        );
        setLocalUser(errorFallbackLocalUser);

        const errorFallbackOpponentUser = {
          id: "opponent",
          username: "Opponent",
          elo: 1500,
          avatarUrl: "https://avatar.iran.liara.run/public/31",
          displayName: "Opponent",
        };
        console.log(
          "Setting error fallback opponent user:",
          errorFallbackOpponentUser
        );
        setOpponentUser(errorFallbackOpponentUser);
      }
    } finally {
      setIsLoading(false);
    }
  };

  // Initialize WebSocket, RTCPeerConnection, and media
  useEffect(() => {
    const token = getAuthToken();
    if (!token || !roomId) return;

    const ws = new WebSocket(
      `ws://localhost:1313/ws?room=${roomId}&token=${token}`
    );
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("WebSocket connected");
      ws.send(JSON.stringify({ type: "join", room: roomId }));
      fetchRoomParticipants();
      getMedia();
    };

    ws.onmessage = async (event) => {
      const data: WSMessage = JSON.parse(event.data);
      console.log("Received WebSocket message:", data);
      switch (data.type) {
        case "topicChange":
          console.log("Received topic change:", data.topic);
          if (data.topic !== undefined) setTopic(data.topic);
          break;
        case "roleSelection":
          console.log("Received role selection:", data.role);
          if (data.role) setPeerRole(data.role);
          break;
        case "ready":
          if (data.ready !== undefined) setPeerReady(data.ready);
          break;
        case "phaseChange":
          if (data.phase) {
            console.log(
              `Received phase change to ${data.phase}. Local role: ${localRole}`
            );
            setDebatePhase(data.phase);
          }
          break;
        case "message":
          if (data.message && peerRole) {
            // Store in speech transcripts for the current phase
            setSpeechTranscripts((prev) => ({
              ...prev,
              [debatePhase]: (prev[debatePhase] || "") + " " + data.message,
            }));
          }
          break;
        case "autoMuteStatus":
          if (data.userId === currentUser?.id) {
            setIsAutoMuted(data.isMuted || false);

            // Automatically mute/unmute microphone based on turn
            if (localStream) {
              const audioTrack = localStream.getAudioTracks()[0];
              if (audioTrack) {
                audioTrack.enabled = !data.isMuted;
              }
            }
          }
          break;
        case "speechText":
          if (data.userId && data.speechText) {
            console.log("Received speech text from backend:", data);
            // Store speech text in transcripts for the current phase
            setSpeechTranscripts((prev) => {
              const updated = {
                ...prev,
                [debatePhase]:
                  (prev[debatePhase] || "") + " " + data.speechText,
              };
              console.log("Updated speech transcripts:", updated);
              return updated;
            });
          }
          break;
        case "userDetails":
          if (data.userDetails) {
            if (data.userDetails.id === currentUser?.id) {
              setLocalUser(data.userDetails);
            } else {
              setOpponentUser(data.userDetails);
            }
          }
          break;
        case "roomParticipants":
          if (data.roomParticipants) {
            console.log(
              "Received room participants update:",
              data.roomParticipants
            );
            setRoomParticipants(data.roomParticipants);
            // Update local and opponent user details when participants change
            if (currentUser && data.roomParticipants.length === 2) {
              const localParticipant = data.roomParticipants.find(
                (p: UserDetails) => p.id === currentUser.id
              );
              const opponentParticipant = data.roomParticipants.find(
                (p: UserDetails) => p.id !== currentUser.id
              );

              console.log("WS - Local participant:", localParticipant);
              console.log("WS - Opponent participant:", opponentParticipant);

              if (localParticipant) {
                setLocalUser({
                  ...localParticipant,
                  avatarUrl:
                    currentUser.avatarUrl || localParticipant.avatarUrl,
                  displayName:
                    currentUser.displayName || localParticipant.displayName,
                });
              }

              if (opponentParticipant) {
                console.log("WS - Setting opponent user:", opponentParticipant);
                setOpponentUser(opponentParticipant);
              }
            }
          }
          break;
        case "offer":
          if (pcRef.current) {
            await pcRef.current.setRemoteDescription(data.offer!);
            const answer = await pcRef.current.createAnswer();
            await pcRef.current.setLocalDescription(answer);
            wsRef.current?.send(JSON.stringify({ type: "answer", answer }));
          }
          break;
        case "answer":
          if (pcRef.current)
            await pcRef.current.setRemoteDescription(data.answer!);
          break;
        case "candidate":
          if (pcRef.current)
            await pcRef.current.addIceCandidate(data.candidate!);
          break;
      }
    };

    ws.onerror = (err) => console.error("WebSocket error:", err);
    ws.onclose = () => console.log("WebSocket closed");

    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
    });
    pcRef.current = pc;

    pc.onicecandidate = (event) => {
      if (event.candidate && wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({ type: "candidate", candidate: event.candidate })
        );
      }
    };

    pc.ontrack = (event) => {
      setRemoteStream(event.streams[0]);
    };

    const getMedia = async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { width: 1280, height: 720 },
          audio: true,
        });
        setLocalStream(stream);
        stream.getTracks().forEach((track) => pc.addTrack(track, stream));
      } catch (err) {
        setMediaError(
          "Failed to access camera/microphone. Please check permissions."
        );
        console.error("Media error:", err);
      }
    };

    return () => {
      if (localStream) localStream.getTracks().forEach((track) => track.stop());
      ws.close();
      pc.close();
    };
  }, [roomId]);

  // Attach streams to video elements
  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream;
      localVideoRef.current
        .play()
        .catch((err) => console.error("Error playing local video:", err));
    }
    if (remoteVideoRef.current && remoteStream) {
      remoteVideoRef.current.srcObject = remoteStream;
      remoteVideoRef.current
        .play()
        .catch((err) => console.error("Error playing remote video:", err));
    }
  }, [localStream, remoteStream]);

  // Initialize Audio Recording
  useEffect(() => {
    const initializeAudio = async () => {
      try {
        // Request microphone access
        const stream = await navigator.mediaDevices.getUserMedia({
          audio: {
            echoCancellation: true,
            noiseSuppression: true,
            autoGainControl: true,
          },
        });

        setAudioStream(stream);
        setAudioError(null);

        // Create audio context for visualization
        const AudioContextClass =
          window.AudioContext ||
          (
            window as typeof window & {
              webkitAudioContext: typeof AudioContext;
            }
          ).webkitAudioContext;
        const audioContext = new AudioContextClass();
        audioContextRef.current = audioContext;

        const source = audioContext.createMediaStreamSource(stream);
        const analyser = audioContext.createAnalyser();
        analyser.fftSize = 256;
        analyser.smoothingTimeConstant = 0.8;

        source.connect(analyser);
        analyserRef.current = analyser;

        // Create media recorder
        const recorder = new MediaRecorder(stream, {
          mimeType: MediaRecorder.isTypeSupported("audio/webm")
            ? "audio/webm"
            : "audio/ogg",
        });

        recorder.ondataavailable = (event) => {
          if (event.data.size > 0 && isMyTurn && localRole) {
            // Here you could send audio data to backend for processing
            // For now, we'll just indicate that audio is being captured
            console.log("Audio data captured:", event.data.size, "bytes");

            // Send indicator that user is speaking
            if (wsRef.current?.readyState === WebSocket.OPEN) {
              wsRef.current.send(
                JSON.stringify({
                  type: "speaking",
                  userId: currentUser?.id,
                  isSpeaking: true,
                })
              );
            }
          }
        };

        recorder.onstart = () => {
          console.log("Audio recording started");
          setIsRecording(true);
        };

        recorder.onstop = () => {
          console.log("Audio recording stopped");
          setIsRecording(false);

          // Send indicator that user stopped speaking
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(
              JSON.stringify({
                type: "speaking",
                userId: currentUser?.id,
                isSpeaking: false,
              })
            );
          }
        };

        setMediaRecorder(recorder);

        console.log("Audio system initialized successfully");
      } catch (error) {
        console.error("Error initializing audio:", error);
        if (error instanceof Error) {
          if (error.name === "NotAllowedError") {
            setAudioError(
              "Microphone access denied. Please allow microphone access and refresh the page."
            );
          } else if (error.name === "NotFoundError") {
            setAudioError(
              "No microphone found. Please connect a microphone and refresh the page."
            );
          } else {
            setAudioError(`Audio initialization failed: ${error.message}`);
          }
        }
      }
    };

    initializeAudio();

    return () => {
      // Cleanup
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
      if (audioContextRef.current) {
        audioContextRef.current.close();
      }
      if (audioStream) {
        audioStream.getTracks().forEach((track) => track.stop());
      }
    };
  }, []);

  // Initialize Speech Recognition
  useEffect(() => {
    const initializeSpeechRecognition = () => {
      if (
        "SpeechRecognition" in window ||
        "webkitSpeechRecognition" in window
      ) {
        const SpeechRecognition =
          window.SpeechRecognition || window.webkitSpeechRecognition;
        if (!SpeechRecognition) {
          setSpeechError("Speech recognition not available");
          return;
        }
        const recognition = new SpeechRecognition();
        recognitionRef.current = recognition;

        recognition.continuous = true;
        recognition.interimResults = true;
        recognition.lang = "en-US";
        recognition.maxAlternatives = 3;

        recognition.onstart = () => {
          console.log("Speech recognition started for phase:", debatePhase);
          setIsListening(true);
          setSpeechError(null);
        };

        recognition.onresult = (event: SpeechRecognitionEvent) => {
          let interimTranscript = "";
          let finalTranscript = "";

          for (let i = event.resultIndex; i < event.results.length; i++) {
            const result = event.results[i];
            if (result.isFinal) {
              finalTranscript += result[0].transcript + " ";
            } else {
              interimTranscript += result[0].transcript;
            }
          }

          if (finalTranscript) {
            // Add final transcript to current phase
            setSpeechTranscripts((prev) => ({
              ...prev,
              [debatePhase]: (prev[debatePhase] || "") + " " + finalTranscript,
            }));
            setCurrentTranscript("");
            console.log("Final transcript:", finalTranscript);
          }
          if (interimTranscript) {
            setCurrentTranscript(interimTranscript);
            console.log("Interim transcript:", interimTranscript);
          }
        };

        recognition.onend = () => {
          console.log("Speech recognition ended");
          setIsListening(false);

          // Restart speech recognition if it's still the user's turn
          if (
            isMyTurn &&
            debatePhase !== DebatePhase.Setup &&
            debatePhase !== DebatePhase.Finished &&
            !isAutoMuted
          ) {
            setTimeout(() => {
              if (recognitionRef.current) {
                try {
                  recognitionRef.current.start();
                  console.log("Speech recognition restarted automatically");
                } catch (error) {
                  console.error("Error restarting speech recognition:", error);
                }
              }
            }, 100);
          }
        };

        recognition.onerror = (event: Event) => {
          const errorEvent = event as unknown as { error: string };
          console.error("Speech recognition error:", errorEvent.error);
          setIsListening(false);
          setSpeechError(`Speech recognition error: ${errorEvent.error}`);
        };

        console.log("Speech recognition initialized");
      } else {
        setSpeechError("Speech recognition not supported in this browser");
      }
    };

    initializeSpeechRecognition();

    return () => {
      if (recognitionRef.current) {
        recognitionRef.current.stop();
      }
    };
  }, [debatePhase]);

  // Audio level monitoring function
  const monitorAudioLevel = () => {
    if (!analyserRef.current) return;

    const dataArray = new Uint8Array(analyserRef.current.frequencyBinCount);
    analyserRef.current.getByteFrequencyData(dataArray);

    // Calculate average volume
    const average =
      dataArray.reduce((sum, value) => sum + value, 0) / dataArray.length;
    const normalizedLevel = Math.min(average / 128, 1); // Normalize to 0-1

    setAudioLevel(normalizedLevel);

    // Continue monitoring
    animationRef.current = requestAnimationFrame(monitorAudioLevel);
  };

  // Start/stop audio recording based on turn
  const startAudioRecording = useCallback(() => {
    if (mediaRecorder && mediaRecorder.state === "inactive") {
      try {
        mediaRecorder.start(1000); // Record in 1-second chunks
        monitorAudioLevel(); // Start audio level monitoring
        setAudioError(null);
      } catch (error) {
        console.error("Error starting recording:", error);
        setAudioError("Failed to start audio recording");
      }
    }
  }, [mediaRecorder]);

  const stopAudioRecording = useCallback(() => {
    if (mediaRecorder && mediaRecorder.state === "recording") {
      try {
        mediaRecorder.stop();
        if (animationRef.current) {
          cancelAnimationFrame(animationRef.current);
        }
        setAudioLevel(0);
      } catch (error) {
        console.error("Error stopping recording:", error);
      }
    }
  }, [mediaRecorder]);

  // Start/stop speech recognition based on turn
  const startSpeechRecognition = useCallback(() => {
    if (recognitionRef.current && !isListening) {
      try {
        recognitionRef.current.start();
        console.log(
          "Speech recognition started for turn in phase:",
          debatePhase
        );
      } catch (error) {
        console.error("Error starting speech recognition:", error);
        setSpeechError("Failed to start speech recognition");
      }
    } else {
      console.log("Cannot start speech recognition:", {
        hasRecognition: !!recognitionRef.current,
        isListening,
        phase: debatePhase,
      });
    }
  }, [isListening, debatePhase]);

  const stopSpeechRecognition = useCallback(() => {
    if (recognitionRef.current && isListening) {
      try {
        recognitionRef.current.stop();
        console.log("Speech recognition stopped for phase:", debatePhase);
      } catch (error) {
        console.error("Error stopping speech recognition:", error);
      }
    } else {
      console.log("Cannot stop speech recognition:", {
        hasRecognition: !!recognitionRef.current,
        isListening,
        phase: debatePhase,
      });
    }
  }, [isListening, debatePhase]);

  // Auto start/stop recording and speech recognition based on turn
  useEffect(() => {
    if (!mediaRecorder || audioError) return;

    if (
      isMyTurn &&
      debatePhase !== DebatePhase.Setup &&
      debatePhase !== DebatePhase.Finished &&
      !isAutoMuted
    ) {
      startAudioRecording();
      startSpeechRecognition();
    } else {
      stopAudioRecording();
      stopSpeechRecognition();
    }

    return () => {
      stopAudioRecording();
      stopSpeechRecognition();
    };
  }, [
    isMyTurn,
    debatePhase,
    isAutoMuted,
    mediaRecorder,
    audioError,
    startAudioRecording,
    stopAudioRecording,
    startSpeechRecognition,
    stopSpeechRecognition,
  ]);

  // Clear audio level when phase changes
  useEffect(() => {
    setAudioLevel(0);
  }, [debatePhase]);

  // Check microphone permissions on component mount
  useEffect(() => {
    const checkPermission = async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({
          audio: true,
        });
        stream.getTracks().forEach((track) => track.stop());
        console.log("Microphone permission granted");
      } catch (error) {
        console.error("Microphone permission denied:", error);
      }
    };
    checkPermission();
  }, []);

  // Handle phase completion
  const handlePhaseDone = () => {
    const currentIndex = phaseOrder.indexOf(debatePhase);
    console.log(
      `handlePhaseDone called for ${localRole}. Current phase: ${debatePhase}, Index: ${currentIndex}`
    );
    if (currentIndex >= 0 && currentIndex < phaseOrder.length - 1) {
      const nextPhase = phaseOrder[currentIndex + 1];
      console.log(
        `Transitioning to next phase: ${nextPhase} for role: ${localRole}`
      );
      setDebatePhase(nextPhase);
      wsRef.current?.send(
        JSON.stringify({ type: "phaseChange", phase: nextPhase })
      );
    } else if (!localRole || !peerRole) {
      console.log("Cannot finish debate: Both roles must be selected.");
      setPopup({
        show: true,
        message: "Both debaters must select roles to finish the debate.",
      });
    } else {
      console.log(`Debate finished for ${localRole}`);
    }
  };

  // Trigger logMessageHistory when debatePhase changes to Finished
  useEffect(() => {
    if (debatePhase === DebatePhase.Finished && localRole) {
      logMessageHistory();
    }
  }, [debatePhase, localRole]);

  // Handlers for user actions
  const handleTopicChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const newTopic = e.target.value;
    setTopic(newTopic);
    const message = JSON.stringify({ type: "topicChange", topic: newTopic });
    console.log("Sending topic change message:", message);
    wsRef.current?.send(message);
  };

  const handleRoleSelection = (role: DebateRole) => {
    if (peerRole === role) {
      alert(
        `Your opponent already chose "${role}". Please select the other side.`
      );
      return;
    }
    setLocalRole(role);
    const message = JSON.stringify({ type: "roleSelection", role });
    console.log("Sending role selection message:", message);
    wsRef.current?.send(message);
    console.log(`Role selected: ${role}`);
  };

  const toggleReady = () => {
    const newReadyState = !localReady;
    setLocalReady(newReadyState);
    wsRef.current?.send(
      JSON.stringify({ type: "ready", ready: newReadyState })
    );
    console.log(`Ready toggled to ${newReadyState} for ${localRole}`);
  };

  // Manage setup popup visibility
  useEffect(() => {
    if (localReady && peerReady) {
      setShowSetupPopup(false);
      setCountdown(3);
      console.log(`Both ready. Starting countdown for ${localRole}`);
    } else {
      setShowSetupPopup(true);
    }
  }, [localReady, peerReady]);

  // Countdown logic
  useEffect(() => {
    if (countdown !== null && countdown > 0) {
      const timer = setTimeout(() => setCountdown(countdown - 1), 1000);
      return () => clearTimeout(timer);
    } else if (countdown === 0) {
      setDebatePhase(DebatePhase.OpeningFor);
      wsRef.current?.send(
        JSON.stringify({ type: "phaseChange", phase: DebatePhase.OpeningFor })
      );
      console.log(
        `Countdown finished. Starting debate at ${DebatePhase.OpeningFor} for ${localRole}`
      );
      if (localRole === "for") {
        pcRef.current
          ?.createOffer()
          .then((offer) =>
            pcRef.current!.setLocalDescription(offer).then(() => offer)
          )
          .then((offer) =>
            wsRef.current?.send(JSON.stringify({ type: "offer", offer }))
          )
          .catch((err) => console.error("Error creating offer:", err));
      }
    }
  }, [countdown, localRole]);

  // Clear input fields on phase change
  useEffect(() => {
    // Clear any audio-related state if needed
    setAudioLevel(0);
  }, [debatePhase]);

  // Debug user state changes
  useEffect(() => {
    console.log("Current user state:", currentUser);
  }, [currentUser]);

  useEffect(() => {
    console.log("Local user state:", localUser);
  }, [localUser]);

  useEffect(() => {
    console.log("Opponent user state:", opponentUser);
  }, [opponentUser]);

  const formatTime = (seconds: number) => {
    const timeStr = `${Math.floor(seconds / 60)}:${(seconds % 60)
      .toString()
      .padStart(2, "0")}`;
    return (
      <span
        className={`font-mono ${
          seconds <= 5 ? "text-red-500 animate-pulse" : "text-gray-600"
        }`}
      >
        {timeStr}
      </span>
    );
  };

  // Render UI
  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-200 p-4">
      <div className="w-full max-w-5xl mx-auto py-2">
        <div className="bg-gradient-to-r from-orange-100 via-white to-orange-100 rounded-xl p-4 text-center">
          <h1 className="text-3xl font-bold text-gray-900">
            Debate: {topic || "No topic set"}
          </h1>
          <p className="mt-2 text-sm text-gray-700">
            Phase: <span className="font-medium">{debatePhase}</span> |
            Participants:{" "}
            <span className="font-medium">{roomParticipants.length}/2</span> |
            Current Turn:{" "}
            <span className="font-semibold text-orange-600">
              {isMyTurn ? "You" : "Opponent"} to{" "}
              {debatePhase.includes("Question")
                ? "ask a question"
                : debatePhase.includes("Answer")
                ? "answer"
                : "make a statement"}
            </span>
            {isAutoMuted && (
              <span className="ml-2 text-red-500 font-medium">
                🔇 Auto-muted (not your turn)
              </span>
            )}
          </p>
        </div>
      </div>

      {/* Setup Popup */}
      {showSetupPopup && (
        <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
          <div className="bg-card text-foreground p-6 rounded-lg shadow-lg max-w-md w-full">
            {/* Header with title and close icon */}
            <div className="flex justify-between items-center mb-6">
              <h2 className="text-2xl font-bold">Debate Setup</h2>
            </div>

            {/* Loading State */}
            {isLoading && (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-t-4 border-primary"></div>
                <span className="ml-2 text-sm text-muted-foreground">
                  Loading participants...
                </span>
              </div>
            )}

            {!isLoading && (
              <>
                {/* Debate Topic */}
                <div className="mb-6">
                  <label className="block text-lg mb-2">Debate Topic</label>
                  <select
                    value={topic}
                    onChange={(e) => handleTopicChange(e)}
                    className="border border-border rounded p-2 w-full bg-input text-foreground mb-2"
                  >
                    <option value="">Select a topic or enter custom</option>
                    {predefinedTopics.map((predefinedTopic, index) => (
                      <option key={index} value={predefinedTopic}>
                        {predefinedTopic}
                      </option>
                    ))}
                  </select>
                  <input
                    type="text"
                    value={topic}
                    onChange={handleTopicChange}
                    placeholder="Or enter a custom debate topic"
                    className="border border-border rounded p-2 w-full bg-input text-foreground"
                  />
                </div>
                {/* Avatars and Role Selection */}
                <div className="mb-6 flex justify-around">
                  {/* Your Avatar and Role Selection */}
                  <div className="flex flex-col items-center">
                    <div className="relative">
                      <img
                        src={
                          localUser?.avatarUrl ||
                          currentUser?.avatarUrl ||
                          "https://avatar.iran.liara.run/public/40"
                        }
                        alt="You"
                        className="w-20 h-20 rounded-full object-cover"
                      />
                      <div
                        className={`absolute top-0 right-0 w-4 h-4 rounded-full border-2 border-card ${
                          localReady ? "bg-green-500" : "bg-red-500"
                        }`}
                        title={
                          localReady ? "You are Ready" : "You are Not Ready"
                        }
                      ></div>
                    </div>
                    <div className="mt-2 text-center">
                      <div className="text-sm font-medium">
                        {localUser?.displayName ||
                          currentUser?.displayName ||
                          "You"}
                      </div>
                      <div className="text-xs text-gray-500">
                        Rating: {localUser?.elo || currentUser?.rating || 1500}
                      </div>
                    </div>
                    <div className="mt-2 flex space-x-2">
                      <button
                        onClick={() => handleRoleSelection("for")}
                        className={`px-2 py-1 rounded text-xs border transition ${
                          localRole === "for"
                            ? "bg-primary text-primary-foreground border-transparent"
                            : "bg-muted text-muted-foreground border-border"
                        }`}
                      >
                        For
                      </button>
                      <button
                        onClick={() => handleRoleSelection("against")}
                        className={`px-2 py-1 rounded text-xs border transition ${
                          localRole === "against"
                            ? "bg-primary text-primary-foreground border-transparent"
                            : "bg-muted text-muted-foreground border-border"
                        }`}
                      >
                        Against
                      </button>
                    </div>
                    <div className="mt-1 text-xs">
                      {localRole
                        ? localRole === "for"
                          ? "For"
                          : "Against"
                        : "Not selected"}
                    </div>
                  </div>
                  {/* Opponent Avatar */}
                  <div className="flex flex-col items-center">
                    <div className="relative">
                      <img
                        src={
                          opponentUser?.avatarUrl ||
                          "https://avatar.iran.liara.run/public/31"
                        }
                        alt="Opponent"
                        className="w-20 h-20 rounded-full object-cover"
                      />
                      <div
                        className={`absolute top-0 right-0 w-4 h-4 rounded-full border-2 border-card ${
                          peerReady ? "bg-green-500" : "bg-red-500"
                        }`}
                        title={
                          peerReady ? "Opponent Ready" : "Opponent Not Ready"
                        }
                      ></div>
                    </div>
                    <div className="mt-2 text-center">
                      <div className="text-sm font-medium">
                        {opponentUser?.displayName ||
                          opponentUser?.username ||
                          (roomParticipants.length > 1
                            ? "Opponent"
                            : "Waiting for opponent...")}
                      </div>
                      <div className="text-xs text-gray-500">
                        Rating: {opponentUser?.elo || 1500}
                      </div>
                      {!opponentUser && roomParticipants.length === 1 && (
                        <div className="text-xs text-orange-500 mt-1">
                          Waiting for opponent to join...
                        </div>
                      )}
                    </div>
                    <div className="mt-2 text-xs">
                      {peerRole
                        ? peerRole === "for"
                          ? "For"
                          : "Against"
                        : "Not selected"}
                    </div>
                  </div>
                </div>
                {/* Ready Button */}
                <div>
                  <Button
                    onClick={toggleReady}
                    className={`w-full py-2 rounded-lg transition ${
                      localReady
                        ? "bg-destructive text-destructive-foreground"
                        : "bg-accent text-accent-foreground"
                    }`}
                  >
                    {localReady ? "Cancel Ready" : "I'm Ready"}
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Countdown Popup */}
      {countdown !== null && countdown > 0 && (
        <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
          <div className="bg-white p-6 rounded-lg shadow-lg text-center">
            <h2 className="text-3xl font-bold">
              Debate starting in {countdown}
            </h2>
          </div>
        </div>
      )}

      {/* Judging Popup */}
      {popup.show && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-70">
          <div className="bg-white rounded-xl shadow-2xl p-6 max-w-md w-full transform transition-all duration-300 scale-105 border border-orange-200">
            {popup.isJudging ? (
              <div className="flex flex-col items-center">
                <div className="animate-spin rounded-full h-16 w-16 border-t-4 border-primary mb-4"></div>
                <h2 className="text-xl font-semibold text-gray-800">
                  {popup.message}
                </h2>
              </div>
            ) : (
              <>
                <h3 className="text-xl font-bold text-orange-600 mb-2">
                  Phase Transition
                </h3>
                <p className="text-gray-700 text-center text-sm">
                  {popup.message}
                </p>
              </>
            )}
          </div>
        </div>
      )}

      {/* Judgment Popup */}
      {showJudgment && judgmentData && (
        <JudgmentPopup
          judgment={judgmentData}
          forRole={localRole === "for" ? "You" : "Opponent"}
          againstRole={localRole === "against" ? "You" : "Opponent"}
          onClose={() => setShowJudgment(false)}
        />
      )}

      <div className="w-full max-w-5xl mx-auto flex flex-col md:flex-row gap-3">
        {/* Local User Section */}
        <div
          className={`relative w-full md:w-1/2 ${
            isMyTurn && debatePhase !== DebatePhase.Finished
              ? "animate-glow"
              : ""
          } bg-white border border-gray-200 shadow-md h-[540px] flex flex-col`}
        >
          <div className="p-2 bg-gray-50 flex items-center gap-2">
            <div className="w-12 h-12 flex-shrink-0">
              <img
                src={
                  localUser?.avatarUrl ||
                  currentUser?.avatarUrl ||
                  "https://avatar.iran.liara.run/public/40"
                }
                alt="You"
                className="w-full h-full rounded-full border border-orange-400 object-cover"
              />
            </div>
            <div className="flex flex-col">
              <div className="text-sm font-medium text-gray-800">
                {localUser?.displayName || currentUser?.displayName || "You"}
              </div>
              <div className="text-xs text-gray-500">
                Role: {localRole || "Not selected"} | Rating:{" "}
                {localUser?.elo || currentUser?.rating || 1500}
              </div>
            </div>
          </div>
          <div className="p-3 flex-1 overflow-y-auto">
            <p className="text-sm font-semibold text-orange-600 mb-1">
              Stance: {localRole}
            </p>
            <p className="text-xs mb-1">
              Time:{" "}
              {formatTime(isMyTurn ? timer : phaseDurations[debatePhase] || 0)}
            </p>
            <video
              ref={localVideoRef}
              autoPlay
              muted
              playsInline
              className="w-full h-80 object-cover"
            />
            {/* Audio and Speech Status */}
            <div className="mt-3 flex items-center justify-center">
              <div className="text-center">
                <div className="text-sm text-gray-600 mb-2">
                  {isRecording && isListening ? (
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 bg-green-500 rounded-full animate-pulse"></div>
                      Recording & Speech Recognition Active
                    </div>
                  ) : isMyTurn &&
                    debatePhase !== DebatePhase.Setup &&
                    debatePhase !== DebatePhase.Finished &&
                    !isAutoMuted ? (
                    <div className="text-blue-600">
                      Will start automatically on your turn
                    </div>
                  ) : (
                    "Inactive"
                  )}
                </div>

                {speechError && (
                  <div className="text-sm text-red-600 p-2 bg-red-50 rounded mb-2">
                    Speech Error: {speechError}
                  </div>
                )}

                {/* Live Transcript Display */}
                {currentTranscript && (
                  <div className="text-sm text-gray-700 p-2 bg-blue-50 rounded mb-2">
                    <div className="font-medium">Live transcript:</div>
                    <div className="italic text-blue-600">
                      {currentTranscript}
                    </div>
                  </div>
                )}

                {/* Current Phase Transcript */}
                {speechTranscripts[debatePhase] && (
                  <div className="text-sm text-gray-700 p-2 bg-green-50 rounded mb-2">
                    <div className="font-medium">Current phase transcript:</div>
                    <div className="text-green-700">
                      {speechTranscripts[debatePhase]}
                    </div>
                  </div>
                )}

                {/* Debug Information */}
                {debugMode && (
                  <div className="text-sm text-gray-700 p-2 bg-yellow-50 rounded mb-2">
                    <div className="font-medium">Debug Info:</div>
                    <div className="text-xs">
                      <div>Phase: {debatePhase}</div>
                      <div>Is My Turn: {isMyTurn ? "Yes" : "No"}</div>
                      <div>Is Listening: {isListening ? "Yes" : "No"}</div>
                      <div>Is Recording: {isRecording ? "Yes" : "No"}</div>
                      <div>Auto Muted: {isAutoMuted ? "Yes" : "No"}</div>
                      <div>
                        Has Recognition: {recognitionRef.current ? "Yes" : "No"}
                      </div>
                    </div>
                  </div>
                )}

                {/* Manual Start Button */}
                {isMyTurn &&
                  debatePhase !== DebatePhase.Setup &&
                  debatePhase !== DebatePhase.Finished &&
                  !isAutoMuted &&
                  !isRecording && (
                    <Button
                      onClick={startAudioRecording}
                      className="bg-green-500 hover:bg-green-600 text-white text-sm px-4 py-2 rounded"
                    >
                      Start Recording
                    </Button>
                  )}

                {/* Stop Button */}
                {isRecording && (
                  <Button
                    onClick={stopAudioRecording}
                    className="bg-red-500 hover:bg-red-600 text-white text-sm px-4 py-2 rounded"
                  >
                    Stop Recording
                  </Button>
                )}

                {/* Manual Speech Recognition Toggle */}
                {isMyTurn &&
                  debatePhase !== DebatePhase.Setup &&
                  debatePhase !== DebatePhase.Finished &&
                  !isAutoMuted && (
                    <div className="mt-2 space-y-2">
                      {!isListening ? (
                        <Button
                          onClick={startSpeechRecognition}
                          className="bg-purple-500 hover:bg-purple-600 text-white text-sm px-4 py-2 rounded"
                        >
                          Start Speech Recognition
                        </Button>
                      ) : (
                        <Button
                          onClick={stopSpeechRecognition}
                          className="bg-purple-500 hover:bg-purple-600 text-white text-sm px-4 py-2 rounded"
                        >
                          Stop Speech Recognition
                        </Button>
                      )}

                      {/* Debug Toggle */}
                      <Button
                        onClick={() => setDebugMode(!debugMode)}
                        className="bg-gray-500 hover:bg-gray-600 text-white text-sm px-4 py-2 rounded"
                      >
                        {debugMode ? "Hide Debug" : "Show Debug"}
                      </Button>
                    </div>
                  )}
              </div>
            </div>

            {/* Audio Recording Status */}
            {(isRecording || audioError) && (
              <div className="mt-3 p-3 bg-gray-50 rounded">
                <div className="text-sm font-medium text-gray-700 mb-2">
                  Audio Status:
                </div>
                {audioError && (
                  <div className="text-sm text-red-600 p-2 bg-red-50 rounded mb-2">
                    Error: {audioError}
                  </div>
                )}
                {isRecording && (
                  <div className="space-y-2">
                    <div className="text-sm text-green-600 p-2 bg-green-50 rounded flex items-center">
                      <div className="w-2 h-2 bg-green-600 rounded-full mr-2 animate-pulse"></div>
                      Recording audio...
                    </div>
                    {/* Audio Level Visualizer */}
                    <div className="bg-gray-200 rounded-full h-2 overflow-hidden">
                      <div
                        className="bg-green-500 h-full transition-all duration-100"
                        style={{ width: `${audioLevel * 100}%` }}
                      ></div>
                    </div>
                    <div className="text-xs text-gray-500">
                      Audio Level: {Math.round(audioLevel * 100)}%
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Remote User Section */}
        <div
          className={`relative w-full md:w-1/2 ${
            !isMyTurn && debatePhase !== DebatePhase.Finished
              ? "animate-glow"
              : ""
          } bg-white border border-gray-200 shadow-md h-[540px] flex flex-col`}
        >
          <div className="p-2 bg-gray-50 flex items-center gap-2">
            <div className="w-12 h-12 flex-shrink-0">
              <img
                src={
                  opponentUser?.avatarUrl ||
                  "https://avatar.iran.liara.run/public/31"
                }
                alt="Opponent"
                className="w-full h-full rounded-full border border-orange-400 object-cover"
              />
            </div>
            <div className="flex flex-col">
              <div className="text-sm font-medium text-gray-800">
                {opponentUser?.displayName ||
                  opponentUser?.username ||
                  "Opponent"}
              </div>
              <div className="text-xs text-gray-500">
                Role: {peerRole || "Not selected"} | Rating:{" "}
                {opponentUser?.elo || 1500}
              </div>
            </div>
          </div>
          <div className="p-3 flex-1 overflow-y-auto">
            <p className="text-sm font-semibold text-orange-600 mb-1">
              Stance: {peerRole}
            </p>
            <p className="text-xs mb-1">
              Time:{" "}
              {formatTime(!isMyTurn ? timer : phaseDurations[debatePhase] || 0)}
            </p>
            <video
              ref={remoteVideoRef}
              autoPlay
              playsInline
              className="w-full h-80 object-cover"
            />
          </div>
        </div>
      </div>

      {/* Speech Transcripts Section */}
      {debatePhase !== DebatePhase.Setup && (
        <div className="w-full max-w-5xl mx-auto mt-4">
          <SpeechTranscripts
            transcripts={speechTranscripts}
            currentPhase={debatePhase}
          />
        </div>
      )}

      {/* Media Error Display */}
      {mediaError && (
        <p className="text-red-500 mt-4 text-center">{mediaError}</p>
      )}

      <style>{`
        @keyframes glow {
          0% {
            box-shadow: 0 0 5px rgba(255, 149, 0, 0.5);
          }
          50% {
            box-shadow: 0 0 20px rgba(255, 149, 0, 0.8);
          }
          100% {
            box-shadow: 0 0 5px rgba(255, 149, 0, 0.5);
          }
        }
        .animate-glow {
          animation: glow 2s infinite;
        }
      `}</style>
    </div>
  );
};

export default OnlineDebateRoom;
