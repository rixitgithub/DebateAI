import React, { useState } from "react";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../components/ui/tabs";
import RoomBrowser from "../components/RoomBrowser";
import Matchmaking from "../components/Matchmaking";

const PlayOnline: React.FC = () => {
  const [activeTab, setActiveTab] = useState("matchmaking");

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-4xl font-bold text-foreground mb-2">Play Online</h1>
        <p className="text-muted-foreground">
          Find opponents and join live debate rooms
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-2 mb-8">
          <TabsTrigger value="matchmaking" className="text-lg">
            🎯 Smart Matchmaking
          </TabsTrigger>
          <TabsTrigger value="browse" className="text-lg">
            🏠 Browse Rooms
          </TabsTrigger>
        </TabsList>

        <TabsContent value="matchmaking" className="space-y-6">
          <div className="bg-gradient-to-r from-blue-500/10 to-purple-500/10 p-4 rounded-lg border">
            <h3 className="text-lg font-semibold text-foreground mb-2">
              🎯 Elo-Based Matchmaking
            </h3>
            <p className="text-muted-foreground">
              Get automatically matched with opponents of similar skill level.
              Our intelligent system considers your Elo rating and wait time to
              find the best matches.
            </p>
          </div>
          <Matchmaking />
        </TabsContent>

        <TabsContent value="browse" className="space-y-6">
          <div className="bg-gradient-to-r from-green-500/10 to-teal-500/10 p-4 rounded-lg border">
            <h3 className="text-lg font-semibold text-foreground mb-2">
              🏠 Live Room Browser
            </h3>
            <p className="text-muted-foreground">
              Browse all available debate rooms and join any that interest you.
              Perfect for spectating ongoing debates or joining specific rooms.
            </p>
          </div>
          <RoomBrowser />
        </TabsContent>
      </Tabs>

      {/* Quick Stats */}
      <div className="mt-12 grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-card p-6 rounded-lg border">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-primary/10 rounded-lg flex items-center justify-center">
              <span className="text-2xl">🎯</span>
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">Smart</div>
              <div className="text-sm text-muted-foreground">
                Elo-based matching
              </div>
            </div>
          </div>
        </div>

        <div className="bg-card p-6 rounded-lg border">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-green-500/10 rounded-lg flex items-center justify-center">
              <span className="text-2xl">⚡</span>
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">Fast</div>
              <div className="text-sm text-muted-foreground">
                Real-time updates
              </div>
            </div>
          </div>
        </div>

        <div className="bg-card p-6 rounded-lg border">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-purple-500/10 rounded-lg flex items-center justify-center">
              <span className="text-2xl">🛡️</span>
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">Fair</div>
              <div className="text-sm text-muted-foreground">
                Balanced matches
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PlayOnline;
