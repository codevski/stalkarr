import { Routes, Route, Navigate } from "react-router-dom";
import { useEffect, useState } from "react";
import LoginPage from "./pages/LoginPage";
import SetupPage from "./pages/SetupPage";
import DashboardPage from "./pages/DashboardPage";
import SonarrPage from "./pages/SonarrPage";
import StatsPage from "./pages/StatsPage";
import SettingsPage from "./pages/SettingsPage";
import Layout from "./components/Layout";
import { Toaster } from "sonner";
import api from "@/lib/api";

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem("token");
  return token ? <>{children}</> : <Navigate to="/login" replace />;
}

function SetupGuard({ children }: { children: React.ReactNode }) {
  const [checking, setChecking] = useState(true);
  const [configured, setConfigured] = useState(true);

  useEffect(() => {
    api
      .get("/api/setup/status")
      .then((res) => setConfigured(res.data.configured))
      .catch(() => setConfigured(true))
      .finally(() => setChecking(false));
  }, []);

  if (checking) return null;
  if (!configured) return <Navigate to="/setup" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/setup" element={<SetupPage />} />
      <Route
        path="/login"
        element={
          <SetupGuard>
            <LoginPage />
          </SetupGuard>
        }
      />
      <Route
        path="/"
        element={
          <SetupGuard>
            <PrivateRoute>
              <Toaster />
              <Layout />
            </PrivateRoute>
          </SetupGuard>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="sonarr" element={<SonarrPage />} />
        <Route path="stats" element={<StatsPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}
