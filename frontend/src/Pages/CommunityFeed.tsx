import React, { useState, useEffect } from 'react';
import { useUser } from '../hooks/useUser';
import CommentTree from '../components/CommentTree';
import ProfileHover from '../components/ProfileHover';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { MessageCircle, UserPlus, UserCheck } from 'lucide-react';

interface Post {
  id: string;
  transcriptId: string;
  userId: string;
  displayName: string;
  avatarUrl?: string;
  topic: string;
  debateType: string;
  opponent: string;
  result: string;
  commentCount: number;
  createdAt: string;
  isFollowing?: boolean;
}

const CommunityFeed: React.FC = () => {
  const { user } = useUser();
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTranscriptId, setSelectedTranscriptId] = useState<string | null>(null);
  const [creatingPost, setCreatingPost] = useState(false);
  const [followingUsers, setFollowingUsers] = useState<Set<string>>(new Set());
  const baseURL = import.meta.env.VITE_BASE_URL || 'http://localhost:1313';

  useEffect(() => {
    fetchFeed();
  }, []);

  const fetchFeed = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('token');
      const response = await fetch(`${baseURL}/posts/feed`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch feed');
      }

      const data = await response.json();
      
      // Fetch follow status for each post if user is logged in
      if (token && data.posts && user) {
        const postsWithStatus = await Promise.all(
          data.posts.map(async (post: Post) => {
            try {
              // Check follow status - check if current user follows this post author
              // Only check if it's not the current user's post
              let isFollowing = false;
              if (post.userId !== user.id) {
                try {
                  const followResponse = await fetch(`${baseURL}/users/${user.id}/following`, {
                    headers: {
                      Authorization: `Bearer ${token}`,
                    },
                  });
                  if (followResponse.ok) {
                    const followData = await followResponse.json();
                    const following = followData.following || [];
                    isFollowing = following.some((f: any) => f.id === post.userId);
                  }
                } catch (err) {
                  console.error('Error checking follow status:', err);
                }
              }

              return { 
                ...post, 
                isFollowing,
              };
            } catch (err) {
              console.error('Error fetching post status:', err);
              return post;
            }
          })
        );
        setPosts(postsWithStatus);
      } else {
        setPosts(data.posts || []);
      }
      
      setError(null);
    } catch (err) {
      console.error('Error fetching feed:', err);
      setError('Failed to load feed');
    } finally {
      setLoading(false);
    }
  };


  const handleFollow = async (userId: string, isFollowing: boolean) => {
    const token = localStorage.getItem('token');
    if (!token) {
      alert('Please log in to follow users');
      return;
    }

    try {
      const endpoint = isFollowing ? 'DELETE' : 'POST';
      const response = await fetch(`${baseURL}/users/${userId}/follow`, {
        method: endpoint,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `Failed to ${isFollowing ? 'unfollow' : 'follow'} user`);
      }

      // Update follow status in posts
      setPosts(posts.map(post => 
        post.userId === userId 
          ? { ...post, isFollowing: !isFollowing }
          : post
      ));
    } catch (err: any) {
      console.error('Error following user:', err);
      alert(err.message || `Failed to ${isFollowing ? 'unfollow' : 'follow'} user`);
    }
  };

  const toggleComments = (transcriptId: string) => {
    if (selectedTranscriptId === transcriptId) {
      setSelectedTranscriptId(null);
    } else {
      setSelectedTranscriptId(transcriptId);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-3xl font-bold mb-2">Community Feed</h1>
        <p className="text-gray-600">See what the community is debating about</p>
      </div>
      
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {posts.length === 0 ? (
        <Card>
          <CardContent className="text-center py-12">
            <MessageCircle className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <p className="text-gray-500 mb-4">No posts yet.</p>
            <p className="text-sm text-gray-400">
              Create a post from your saved transcripts to get started!
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {posts.map((post) => (
            <Card key={post.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <ProfileHover userId={post.userId}>
                        {post.avatarUrl ? (
                          <img
                            src={post.avatarUrl}
                            alt={post.displayName || 'User'}
                            className="w-10 h-10 rounded-full object-cover border-2 border-gray-200 cursor-pointer hover:border-primary transition-colors"
                            onError={(e) => {
                              const target = e.target as HTMLImageElement;
                              target.style.display = 'none';
                              const parent = target.parentElement;
                              if (parent) {
                                const fallback = document.createElement('div');
                                fallback.className = 'w-10 h-10 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium border-2 border-gray-200 cursor-pointer hover:border-primary transition-colors';
                                fallback.textContent = post.displayName?.charAt(0).toUpperCase() || 'U';
                                parent.appendChild(fallback);
                              }
                            }}
                          />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium border-2 border-gray-200 cursor-pointer hover:border-primary transition-colors">
                            {post.displayName?.charAt(0).toUpperCase() || 'U'}
                          </div>
                        )}
                      </ProfileHover>
                      <div className="flex-1">
                        <CardTitle className="text-xl mb-1">{post.topic}</CardTitle>
                        <div className="flex items-center gap-2 text-sm text-gray-500">
                          <span>by {post.displayName}</span>
                          <span>•</span>
                          <span>{new Date(post.createdAt).toLocaleDateString()}</span>
                        </div>
                      </div>
                      {user && post.userId !== user.id && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleFollow(post.userId, post.isFollowing || false)}
                          className="flex items-center gap-2"
                        >
                          {post.isFollowing ? (
                            <>
                              <UserCheck className="w-4 h-4" />
                              <span>Following</span>
                            </>
                          ) : (
                            <>
                              <UserPlus className="w-4 h-4" />
                              <span>Follow</span>
                            </>
                          )}
                        </Button>
                      )}
                      {user && post.userId === user.id && (
                        <span className="text-xs text-gray-500 px-2">Your Post</span>
                      )}
                    </div>
                  </div>
                  <span className={`px-3 py-1 text-xs rounded-full font-medium ${
                    post.result === 'win' 
                      ? 'bg-green-100 text-green-800' 
                      : post.result === 'loss'
                      ? 'bg-red-100 text-red-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {post.result}
                  </span>
                </div>
              </CardHeader>
              
              <CardContent>
                <div className="mb-4 space-y-2 text-sm">
                  <p><strong>Type:</strong> {post.debateType}</p>
                  <p><strong>Opponent:</strong> {post.opponent}</p>
                </div>

                <div className="flex items-center gap-4 pt-4 border-t">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => toggleComments(post.transcriptId)}
                    className="flex items-center gap-2"
                  >
                    <MessageCircle className="w-4 h-4" />
                    <span>{post.commentCount}</span>
                  </Button>
                </div>

                {selectedTranscriptId === post.transcriptId && (
                  <div className="mt-4 pt-4 border-t">
                    <CommentTree
                      transcriptId={post.transcriptId}
                      onCommentAdded={() => {
                        // Update comment count in local state
                        setPosts(posts.map(p => 
                          p.id === post.id 
                            ? { ...p, commentCount: p.commentCount + 1 }
                            : p
                        ));
                      }}
                    />
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

export default CommunityFeed;
