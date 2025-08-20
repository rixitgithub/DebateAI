import React, { useEffect, useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAtom } from 'jotai';
import Avatar from 'react-avatar';
import { userAtom } from '../state/userAtom';

interface MatchmakingPool {
  userId: string;
  username: string;
  elo: number;
  minElo: number;
  maxElo: number;
  joinedAt: string;
  lastActivity: string;
}

interface MatchmakingMessage {
  type: string;
  userId?: string;
  username?: string;
  elo?: number;
  roomId?: string;
  pool?: string;
  error?: string;
}

const Matchmaking: React.FC = () => {
  const [pool, setPool] = useState<MatchmakingPool[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isInPool, setIsInPool] = useState(false);
  const [waitTime, setWaitTime] = useState(0);
  const [user] = useAtom(userAtom);
  const wsRef = useRef<WebSocket | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    // Check if user is logged in
    if (!user) {
      console.log('No user found, redirecting to login');
      navigate('/login');
      return;
    }

    // Get token from localStorage
    const token = localStorage.getItem('token');
    if (!token) {
      console.log('No token found, redirecting to login');
      navigate('/login');
      return;
    }

    console.log(
      'Connecting to WebSocket with token:',
      token ? 'present' : 'missing'
    );

    // Connect to WebSocket with authentication token
    const ws = new WebSocket(
      `ws://localhost:1313/ws/matchmaking?token=${token}`
    );
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      console.log('✅ Connected to matchmaking WebSocket');
    };

    ws.onmessage = (event) => {
      const message: MatchmakingMessage = JSON.parse(event.data);

      switch (message.type) {
        case 'pool_update':
          if (message.pool) {
            const poolData: MatchmakingPool[] = JSON.parse(message.pool);
            setPool(poolData);

            // Check if current user is in pool
            const currentUser = poolData.find(
              (p) => p.username === user.displayName
            );
            setIsInPool(!!currentUser);
          }
          break;

        case 'room_created':
          if (message.roomId) {
            // Navigate to the created room
            navigate(`/debate-room/${message.roomId}`);
          }
          break;

        default:
          console.log('Received message:', message);
      }
    };

    ws.onclose = (event) => {
      setIsConnected(false);
      console.log(
        '🔌 Disconnected from matchmaking WebSocket:',
        event.code,
        event.reason
      );
    };

    ws.onerror = (error) => {
      console.error('❌ WebSocket error:', error);
      setIsConnected(false);
    };

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [navigate, user]);

  const joinPool = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'join_pool' }));
      setIsInPool(true);
    }
  };

  const leavePool = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'leave_pool' }));
      setIsInPool(false);
    }
  };

  const updateActivity = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'update_activity' }));
    }
  };

  // Update activity every 30 seconds
  useEffect(() => {
    if (isInPool) {
      const interval = setInterval(updateActivity, 30000);
      return () => clearInterval(interval);
    }
  }, [isInPool]);

  // Update wait time
  useEffect(() => {
    if (isInPool) {
      const interval = setInterval(() => {
        setWaitTime((prev) => prev + 1);
      }, 1000);
      return () => clearInterval(interval);
    } else {
      setWaitTime(0);
    }
  }, [isInPool]);

  const formatWaitTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div className='matchmaking p-6 bg-card rounded-lg shadow-2xl max-w-4xl mx-auto'>
      <div className='flex justify-between items-center mb-6'>
        <h2 className='text-3xl font-bold text-foreground'>
          Online Matchmaking
        </h2>
        <div className='flex items-center gap-4'>
          <div className='text-sm text-muted-foreground'>
            Your Elo:{' '}
            <span className='font-semibold text-foreground'>
              {user?.rating || 0}
            </span>
          </div>
          <div
            className={`w-3 h-3 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-red-500'
            }`}
          ></div>
        </div>
      </div>

      {/* Matchmaking Controls */}
      <div className='bg-popover p-4 rounded-lg mb-6'>
        <div className='flex items-center justify-between'>
          <div>
            <h3 className='text-lg font-semibold text-foreground mb-2'>
              Find a Debate Partner
            </h3>
            <p className='text-sm text-muted-foreground'>
              Join the matchmaking pool to find opponents with similar Elo
              ratings
            </p>
          </div>

          {isInPool ? (
            <div className='text-center'>
              <div className='text-2xl font-bold text-primary mb-1'>
                {formatWaitTime(waitTime)}
              </div>
              <div className='text-sm text-muted-foreground'>Searching...</div>
              <button
                onClick={leavePool}
                className='mt-2 bg-destructive text-destructive-foreground px-4 py-2 rounded hover:bg-destructive/90 transition'
              >
                Cancel Search
              </button>
            </div>
          ) : (
            <button
              onClick={joinPool}
              disabled={!isConnected}
              className='bg-primary text-primary-foreground px-6 py-3 rounded-lg hover:bg-primary/90 transition disabled:opacity-50'
            >
              Start Matchmaking
            </button>
          )}
        </div>
      </div>

      {/* Matchmaking Pool */}
      <div className='bg-popover p-4 rounded-lg'>
        <h3 className='text-lg font-semibold text-foreground mb-4'>
          Players in Queue ({pool.length})
        </h3>

        {pool.length === 0 ? (
          <p className='text-center text-muted-foreground py-8'>
            No players currently searching for matches
          </p>
        ) : (
          <div className='grid gap-3'>
            {pool.map((player) => (
              <div
                key={player.userId}
                className='flex items-center justify-between p-3 bg-background rounded-lg border'
              >
                <div className='flex items-center gap-3'>
                  <Avatar
                    name={player.username}
                    size='40'
                    round
                    className='border-2 border-border'
                  />
                  <div>
                    <div className='font-medium text-foreground'>
                      {player.username}
                    </div>
                    <div className='text-sm text-muted-foreground'>
                      Elo: {player.elo} (Range: {player.minElo}-{player.maxElo})
                    </div>
                  </div>
                </div>

                <div className='text-right'>
                  <div className='text-sm text-muted-foreground'>
                    Waiting:{' '}
                    {formatWaitTime(
                      Math.floor(
                        (Date.now() - new Date(player.joinedAt).getTime()) /
                          1000
                      )
                    )}
                  </div>
                  {player.userId === user?.id && (
                    <div className='text-xs text-primary font-medium'>You</div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Matchmaking Info */}
      <div className='mt-6 p-4 bg-muted rounded-lg'>
        <h4 className='font-semibold text-foreground mb-2'>
          How Matchmaking Works
        </h4>
        <ul className='text-sm text-muted-foreground space-y-1'>
          <li>
            • Players are matched based on Elo rating similarity (±200 points by
            default)
          </li>
          <li>
            • Wait time is considered to prioritize players who have been
            waiting longer
          </li>
          <li>
            • Matches are created automatically when compatible opponents are
            found
          </li>
          <li>
            • You'll be automatically redirected to the debate room when matched
          </li>
        </ul>
      </div>
    </div>
  );
};

export default Matchmaking;
