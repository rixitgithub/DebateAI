import React, { useContext } from "react";
import { Routes, Route, Navigate, Outlet } from "react-router-dom";

import { AuthProvider, AuthContext } from "./context/authContext";
import { ThemeProvider } from "./context/theme-provider";

// Pages
import Home from "./Pages/Home";
import Authentication from "./Pages/Authentication";
import DebateApp from "./Pages/Game";
import Profile from "./Pages/Profile";
import Leaderboard from "./Pages/Leaderboard";
import StartDebate from "./Pages/StartDebate";
import About from "./Pages/About";

// Layout
import Layout from "./components/Layout";

// Protects routes based on authentication status
function ProtectedRoute() {
  const authContext = useContext(AuthContext);

  // Throw error if context is undefined (shouldn't happen within AuthProvider)
  if (!authContext) {
    throw new Error("ProtectedRoute must be used within an AuthProvider");
  }

  const { isAuthenticated, loading: isLoading } = authContext;

  if (isLoading) {
    return <div>Loading...</div>;
  }
  return isAuthenticated ? <Outlet /> : <Navigate to="/" replace />;
}

// Defines application routes
function AppRoutes() {
  const authContext = useContext(AuthContext);

  // Throw error if context is undefined (shouldn't happen within AuthProvider)
  if (!authContext) {
    throw new Error("AppRoutes must be used within an AuthProvider");
  }

  const { isAuthenticated } = authContext;

  return (
    <Routes>
      {/* Public routes */}
      <Route
        path="/"
        element={
          isAuthenticated ? (
            <Navigate to="/startDebate" replace />
          ) : (
            <Home />
          )
        }
      />
      <Route path="/auth" element={<Authentication />} />

      {/* Protected routes with layout */}
      <Route element={<ProtectedRoute />}>
        <Route path="/" element={<Layout />}>
          <Route path="startDebate" element={<StartDebate />} />
          <Route path="leaderboard" element={<Leaderboard />} />
          <Route path="profile" element={<Profile />} />
          <Route path="about" element={<About />} />
          <Route path="game/:userId" element={<DebateApp />} />
        </Route>
      </Route>

      {/* Redirect unknown routes */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

// Main app with providers
function App() {
  return (
    <AuthProvider>
      <ThemeProvider>
        <AppRoutes />
      </ThemeProvider>
    </AuthProvider>
  );
}

export default App;
