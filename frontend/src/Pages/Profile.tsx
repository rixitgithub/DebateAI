"use client";

import React, { useState, useEffect, useRef } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/text-area";
import { Separator } from "@/components/ui/separator";
import defaultAvatar from "@/assets/avatar2.jpg";
import {
  CheckCircle,
  XCircle,
  MinusCircle,
  Medal,
  Twitter,
  Users,
  TrendingUp,
  Award,
  Instagram,
  Linkedin,
  Pen,
} from "lucide-react";
import {
  PieChart,
  Pie,
  ResponsiveContainer,
  LineChart,
  Line,
  LabelList,
} from "recharts";
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { getProfile, updateProfile } from "@/services/profileService";
import { getAuthToken } from "@/utils/auth";

interface ProfileData {
  displayName: string;
  email: string;
  bio: string;
  eloRating: number;
  twitter?: string;
  instagram?: string;
  linkedin?: string;
  avatarUrl?: string;
}

interface LeaderboardEntry {
  rank: number;
  name: string;
  score: number;
  avatarUrl: string;
  currentUser?: boolean;
}

interface DebateResult {
  topic: string;
  result: "win" | "loss" | "draw";
  eloChange: number;
}

interface StatData {
  wins: number;
  losses: number;
  draws: number;
  eloHistory: { month: string; elo: number }[];
}

interface DashboardData {
  profile: ProfileData;
  leaderboard: LeaderboardEntry[];
  debateHistory: DebateResult[] | null;
  stats: StatData;
}

const Profile: React.FC = () => {
  const [dashboard, setDashboard] = useState<DashboardData | null>(null);
  const [editingField, setEditingField] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const fetchDashboard = async () => {
      const token = getAuthToken();
      if (!token) {
        setErrorMessage("Please log in to view your profile.");
        setLoading(false);
        return;
      }

      try {
        const data = await getProfile(token);
        setDashboard(data);
      } catch (err) {
        setErrorMessage("Failed to load dashboard data.");
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboard();
  }, []);

  useEffect(() => {
    if (editingField === "displayName" && inputRef.current) {
      inputRef.current.focus();
    }
  }, [editingField]);

  useEffect(() => {
    if (successMessage) {
      const timer = setTimeout(() => {
        setSuccessMessage("");
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [successMessage]);

  const handleSubmit = async (e: React.FormEvent, field: string) => {
    e.preventDefault();
    if (!dashboard?.profile) return;
    const token = getAuthToken();
    if (!token) {
      setErrorMessage("Authentication token is missing.");
      return;
    }
    try {
      await updateProfile(
        token,
        dashboard.profile.displayName,
        dashboard.profile.bio,
        dashboard.profile.twitter,
        dashboard.profile.instagram,
        dashboard.profile.linkedin
      );
      setSuccessMessage(`${field.charAt(0).toUpperCase() + field.slice(1)} updated successfully!`);
      setErrorMessage("");
      setEditingField(null);
    } catch (err) {
      setErrorMessage(`Failed to update ${field}.`);
      console.error(err);
    }
  };

  const renderEditableSocialField = (
    field: keyof ProfileData,
    label: string,
    Icon: React.ComponentType<{ className?: string }>,
    placeholder: string = `Enter your ${label.toLowerCase()}`
  ) => {
    return editingField === field ? (
      <form
        onSubmit={(e) => handleSubmit(e, field as string)}
        className="space-y-2 mb-2"
      >
        <div className="flex items-center gap-2">
          <Icon className="w-5 h-5 text-primary" />
          <Input
            id={field}
            type="text"
            value={dashboard?.profile[field] || ""}
            onChange={(e) =>
              setDashboard({
                ...dashboard!,
                profile: { ...dashboard!.profile, [field]: e.target.value },
              })
            }
            placeholder={placeholder}
            className="mt-1"
          />
        </div>
        <div className="flex gap-2">
          <Button type="submit" size="sm" variant="default" className="flex-1">
            Save
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setEditingField(null)}
            className="flex-1"
          >
            Cancel
          </Button>
        </div>
      </form>
    ) : (
      <div className="flex items-center justify-between mb-2">
        {dashboard?.profile[field] ? (
          <a
            href={
              field === "twitter"
                ? `https://twitter.com/${dashboard.profile[field]}`
                : field === "instagram"
                ? `https://instagram.com/${dashboard.profile[field]}`
                : `https://linkedin.com/in/${dashboard.profile[field]}`
            }
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-primary hover:underline flex items-center gap-2"
          >
            <Icon className="w-5 h-5 text-primary" />
            {field === "twitter" || field === "instagram"
              ? `@${dashboard.profile[field]}`
              : dashboard.profile[field]}
          </a>
        ) : (
          <span className="text-sm text-muted-foreground flex items-center gap-2">
            <Icon className="w-5 h-5 text-muted-foreground" />
            Add {label.toLowerCase()}
          </span>
        )}
        <button
          onClick={() => setEditingField(field as string)}
          className="p-2 hover:bg-muted rounded-full transition-colors"
          title={`Edit ${label}`}
        >
          <Pen className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>
    );
  };

  const renderBioField = () => {
    return editingField === "bio" ? (
      <form
        onSubmit={(e) => handleSubmit(e, "bio")}
        className="space-y-2 mb-2"
      >
        <Label htmlFor="bio">Bio</Label>
        <Textarea
          id="bio"
          value={dashboard?.profile.bio || ""}
          onChange={(e) =>
            setDashboard({
              ...dashboard!,
              profile: { ...dashboard!.profile, bio: e.target.value },
            })
          }
          placeholder="Share your story"
          className="mt-1"
        />
        <div className="flex gap-2">
          <Button type="submit" size="sm" variant="default" className="flex-1">
            Save
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setEditingField(null)}
            className="flex-1"
          >
            Cancel
          </Button>
        </div>
      </form>
    ) : (
      <div className="flex items-start justify-between mb-2">
        <span className="text-sm text-foreground whitespace-pre-wrap">
          {dashboard?.profile.bio || "Add your bio"}
        </span>
        <button
          onClick={() => setEditingField("bio")}
          className="p-2 hover:bg-muted rounded-full transition-colors"
          title="Edit Bio"
        >
          <Pen className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>
    );
  };

  if (loading) {
    return (
      <div className="p-4 flex justify-center items-center h-[calc(100vh-8rem)]">
        <div className="text-center">
          <div className="animate-pulse rounded-full bg-muted h-12 w-12 mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading Profile...</p>
        </div>
      </div>
    );
  }

  if (!dashboard) {
    return (
      <div className="p-4 text-red-500 text-center">{errorMessage}</div>
    );
  }

  const { profile, leaderboard, debateHistory, stats } = dashboard;

  const donutChartData = [
    { label: "Losses", value: stats.losses, fill: "hsl(var(--chart-1))" },
    { label: "Wins", value: stats.wins, fill: "hsl(var(--chart-2))" },
    { label: "Draws", value: stats.draws, fill: "hsl(var(--chart-3))" },
  ];
  const totalMatches = donutChartData.reduce(
    (acc, curr) => acc + curr.value,
    0
  );

  const donutChartConfig: ChartConfig = {
    value: { label: "Matches" },
    wins: { label: "Wins", color: "hsl(var(--chart-2))" },
    losses: { label: "Losses", color: "hsl(var(--chart-1))" },
    draws: { label: "Draws", color: "hsl(var(--chart-3))" },
  };

  const eloChartConfig: ChartConfig = {
    elo: { label: "Elo", color: "hsl(var(--primary))" },
  };

  return (
    <div className="w-full flex flex-col md:flex-row gap-4 p-2 md:p-2 bg-background">
      {/* Left Column: Profile Details */}
      <div className="flex flex-col w-full md:w-[20%] bg-card p-4 border border-border rounded-md shadow md:h-[calc(100vh-8rem)] overflow-auto">
        {successMessage && (
          <div className="mb-2 p-2 rounded bg-green-100 text-green-700 text-sm animate-in fade-in duration-300">
            {successMessage}
          </div>
        )}
        {errorMessage && (
          <div className="mb-2 p-2 rounded bg-red-100 text-red-700 text-sm animate-in fade-in duration-300">
            {errorMessage}
          </div>
        )}
        <div className="flex flex-col items-center mb-4">
          <div className="w-24 h-24 rounded-full overflow-hidden bg-muted flex-shrink-0 mb-2 border-2 border-primary shadow-md hover:shadow-lg transition-shadow">
            <img
              src={profile.avatarUrl || defaultAvatar}
              alt="Avatar"
              className="object-cover w-full h-full"
            />
          </div>
          {editingField === "displayName" ? (
            <form
              onSubmit={(e) => handleSubmit(e, "displayName")}
              className="flex flex-col items-center space-y-2 w-full"
            >
              <Input
                id="displayName"
                type="text"
                value={profile.displayName || ""}
                onChange={(e) =>
                  setDashboard({
                    ...dashboard,
                    profile: { ...profile, displayName: e.target.value },
                  })
                }
                ref={inputRef}
                className="text-2xl font-bold h-10 w-40"
                placeholder="Enter name"
              />
              <div className="flex gap-2 w-full">
                <Button type="submit" size="sm" variant="default" className="flex-1">
                  Save
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setEditingField(null)}
                  className="flex-1"
                >
                  Cancel
                </Button>
              </div>
            </form>
          ) : (
            <div className="flex items-center space-x-2">
              <h2 className="text-2xl font-bold text-foreground">
                {profile.displayName || "Steve"}
              </h2>
              <button
                onClick={() => setEditingField("displayName")}
                className="p-1 hover:bg-muted rounded-full"
                title="Edit Display Name"
              >
                <Pen
                  className={`w-5 h-5 ${profile.displayName ? "text-primary" : "text-muted-foreground"}`}
                />
              </button>
            </div>
          )}
          <p className="text-lg bg-primary text-primary-foreground px-2 py-1 rounded mt-2">
            Elo: {profile.eloRating}
          </p>
        </div>
        <Separator className="my-2" />
        <p className="text-sm text-muted-foreground mb-2">
          Email: {profile.email}
        </p>
        {/* Socials Section */}
        <div className="space-y-2 mb-4">
          <h3 className="text-sm font-semibold text-foreground">Socials</h3>
          {renderEditableSocialField(
            "twitter",
            "X / Twitter",
            Twitter,
            "Your Twitter handle (without @)"
          )}
          {renderEditableSocialField(
            "instagram",
            "Instagram",
            Instagram,
            "Your Instagram handle (without @)"
          )}
          {renderEditableSocialField(
            "linkedin",
            "LinkedIn",
            Linkedin,
            "Your LinkedIn profile (username or ID)"
          )}
        </div>
        <Separator className="my-2" />
        {/* Bio Section */}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-foreground">Bio</h3>
          {renderBioField()}
        </div>
      </div>

      {/* Right Column: Dashboard */}
      <div className="flex-1 flex flex-col space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Donut Chart */}
          <Card className="shadow h-[275px]">
            <CardContent className="flex-1 pb-0">
              {totalMatches === 0 ? (
                <div className="flex flex-col items-center justify-center h-[240px] text-center">
                  <Award className="w-12 h-12 text-muted-foreground mb-2 animate-pulse" />
                  <p className="text-sm text-muted-foreground mb-2">
                    No matches yet!
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => (window.location.href = "/debates")}
                    className="hover:bg-primary hover:text-primary-foreground"
                  >
                    Start Debating
                  </Button>
                </div>
              ) : (
                <ChartContainer
                  config={donutChartConfig}
                  className="mx-auto aspect-square max-h-[240px]"
                >
                  <PieChart>
                    <ChartTooltip
                      cursor={false}
                      content={<ChartTooltipContent hideLabel />}
                    />
                    <Pie
                      data={donutChartData}
                      dataKey="value"
                      nameKey="label"
                      innerRadius={50}
                      strokeWidth={3}
                    >
                      <LabelList
                        content={({ viewBox }) => {
                          if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                            return (
                              <text
                                x={viewBox.cx}
                                y={viewBox.cy}
                                textAnchor="middle"
                                dominantBaseline="middle"
                              >
                                <tspan
                                  x={viewBox.cx}
                                  y={viewBox.cy}
                                  className="fill-foreground text-xl font-bold"
                                >
                                  {totalMatches}
                                </tspan>
                                <tspan
                                  x={viewBox.cx}
                                  y={(viewBox.cy || 0) + 18}
                                  className="fill-muted-foreground text-sm"
                                >
                                  Matches
                                </tspan>
                              </text>
                            );
                          }
                        }}
                      />
                    </Pie>
                  </PieChart>
                </ChartContainer>
              )}
            </CardContent>
          </Card>

          {/* Elo Trend */}
          <Card className="shadow h-[275px] flex flex-col">
            <CardHeader className="p-2 flex-shrink-0">
              <CardTitle className="text-foreground text-lg">Ratings</CardTitle>
            </CardHeader>
            <CardContent className="p-0 flex-1 flex items-center justify-center">
              {stats.eloHistory && stats.eloHistory.length > 0 ? (
                <ChartContainer
                  config={eloChartConfig}
                  className="w-[90%] h-[80%]"
                >
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart
                      data={stats.eloHistory}
                      margin={{ top: 58, right: 12, left: 12, bottom: 8 }}
                    >
                      <ChartTooltip
                        cursor={false}
                        content={<ChartTooltipContent hideLabel />}
                      />
                      <Line
                        dataKey="elo"
                        type="monotone"
                        stroke="hsl(var(--primary))"
                        strokeWidth={2}
                        dot={{ fill: "hsl(var(--primary))" }}
                        activeDot={{ r: 6 }}
                      >
                        <LabelList
                          dataKey="elo"
                          position="top"
                          offset={6}
                          className="fill-foreground text-xs"
                        />
                      </Line>
                    </LineChart>
                  </ResponsiveContainer>
                </ChartContainer>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <TrendingUp className="w-12 h-12 text-muted-foreground mb-2 animate-pulse" />
                  <p className="text-sm text-muted-foreground mb-2">
                    No Elo history yet!
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => (window.location.href = "/debates")}
                    className="hover:bg-primary hover:text-primary-foreground"
                  >
                    Join a Debate
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Bottom row: Leaderboard + Recent Debates */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Mini Leaderboard */}
          <Card className="shadow h-[275px]">
            <CardHeader className="p-2">
              <CardTitle className="text-foreground text-lg">
                Top 5 Debaters
              </CardTitle>
              <CardDescription className="text-muted-foreground text-xs">
                See who’s leading
              </CardDescription>
            </CardHeader>
            <Separator />
            <CardContent className="p-2 overflow-auto">
              {leaderboard && leaderboard.length > 0 ? (
                <ul className="space-y-1">
                  {leaderboard.slice(0, 5).map((leader, index) => (
                    <li
                      key={index}
                      className="flex items-center justify-between border-b border-muted py-1 last:border-none text-sm hover:bg-muted/50 transition-colors"
                    >
                      <span className="font-medium flex items-center space-x-2">
                        <span>{leader.rank}</span>
                        {leader.rank === 1 && (
                          <Medal className="w-3 h-3 text-yellow-500" />
                        )}
                        {leader.rank === 2 && (
                          <Medal className="w-3 h-3 text-gray-400" />
                        )}
                        {leader.rank === 3 && (
                          <Medal className="w-3 h-3 text-amber-600" />
                        )}
                        <img
                          src={leader.avatarUrl}
                          alt={leader.name}
                          className="w-6 h-6 rounded-full"
                        />
                        <span>{leader.name}</span>
                      </span>
                      <span className="text-muted-foreground text-xs">
                        {leader.score}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <Users className="w-12 h-12 text-muted-foreground mb-2 animate-pulse" />
                  <p className="text-sm text-muted-foreground mb-2">
                    Leaderboard is empty!
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Check back later to see top debaters.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recent Debates */}
          <Card className="shadow h-[275px]">
            <CardHeader className="p-2">
              <CardTitle className="text-foreground text-lg">
                Recent Debates
              </CardTitle>
              <CardDescription className="text-muted-foreground text-xs">
                Win/Loss record & Elo changes
              </CardDescription>
            </CardHeader>
            <Separator />
            <CardContent className="p-2 overflow-auto">
              {debateHistory && debateHistory.length > 0 ? (
                <ul className="space-y-1">
                  {debateHistory.map((debate, idx) => {
                    const IconComponent =
                      debate.result === "win"
                        ? CheckCircle
                        : debate.result === "loss"
                        ? XCircle
                        : MinusCircle;
                    const iconColor =
                      debate.result === "win"
                        ? "text-green-600"
                        : debate.result === "loss"
                        ? "text-red-600"
                        : "text-gray-600";
                    return (
                      <li
                        key={idx}
                        className="flex items-center justify-between border-b border-muted py-1 last:border-none text-sm hover:bg-muted/50 transition-colors"
                      >
                        <span className="font-medium flex items-center">
                          <IconComponent
                            className={`w-3 h-3 mr-1 ${iconColor}`}
                          />
                          {debate.topic}
                        </span>
                        <span className={`${iconColor} font-semibold text-xs`}>
                          {debate.result.toUpperCase()}{" "}
                          {debate.eloChange > 0 && `(+${debate.eloChange})`}
                          {debate.eloChange < 0 && `(${debate.eloChange})`}
                        </span>
                      </li>
                    );
                  })}
                </ul>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <Award className="w-12 h-12 text-muted-foreground mb-2 animate-pulse" />
                  <p className="text-sm text-muted-foreground mb-2">
                    No recent debates available.
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => (window.location.href = "/debates")}
                    className="hover:bg-primary hover:text-primary-foreground"
                  >
                    Join a Debate
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Profile;