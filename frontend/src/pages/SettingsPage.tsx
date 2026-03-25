import { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Tv,
  Plus,
  Trash2,
  Save,
  CheckCircle2,
  Loader2,
  Pencil,
  Wifi,
  AlertCircle,
} from "lucide-react";
import api from "@/lib/api";
import type { SonarrInstance } from "@/types";
import { KeyRound } from "lucide-react";
import axios from "axios";
import { cn } from "@/lib/utils";

interface InstanceFormProps {
  initial?: SonarrInstance;
  onSave: (data: {
    name: string;
    url: string;
    api_key: string;
  }) => Promise<void>;
  onCancel?: () => void;
  saveLabel?: string;
}

function InstanceForm({
  initial,
  onSave,
  onCancel,
  saveLabel = "Save",
}: InstanceFormProps) {
  const [name, setName] = useState(initial?.name ?? "");
  const [url, setUrl] = useState(initial?.url ?? "");
  const [apiKey, setApiKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    message: string;
  } | null>(null);

  async function handleTest() {
    if (!initial?.id) return;
    setTesting(true);
    setTestResult(null);
    try {
      const res = await api.post(`/api/settings/sonarr/${initial.id}/test`);
      setTestResult({
        ok: true,
        message: `Connected — Sonarr v${res.data.version}`,
      });
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setTestResult({
          ok: false,
          message: err.response?.data?.error ?? "Could not reach Sonarr",
        });
      } else {
        setTestResult({ ok: false, message: "Could not reach Sonarr" });
      }
    } finally {
      setTesting(false);
    }
  }

  async function handleSave() {
    setSaving(true);
    setSaved(false);
    setTestResult(null);
    try {
      await onSave({ name, url, api_key: apiKey });
      setSaved(true);
      setApiKey("");
      setTimeout(() => setSaved(false), 3000);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="grid sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1.5">
          <Label>Name</Label>
          <Input
            placeholder="Sonarr 4K"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>URL</Label>
          <Input
            placeholder="http://localhost:8989"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
          />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>
          API Key
          {initial?.api_key_set && (
            <span className="ml-2 text-xs text-muted-foreground font-normal">
              current: <span className="font-mono">{initial.api_key_hint}</span>
            </span>
          )}
        </Label>
        <Input
          type="password"
          placeholder={
            initial?.api_key_set
              ? "Leave blank to keep current key"
              : "Enter API key"
          }
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
        />
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <Button size="sm" onClick={handleSave} disabled={saving}>
          {saving ? (
            <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
          ) : (
            <Save className="w-3.5 h-3.5 mr-1.5" />
          )}
          {saveLabel}
        </Button>

        {initial?.id && (
          <Button
            size="sm"
            variant="outline"
            onClick={handleTest}
            disabled={testing}
          >
            {testing ? (
              <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
            ) : (
              <Wifi className="w-3.5 h-3.5 mr-1.5" />
            )}
            Test
          </Button>
        )}

        {onCancel && (
          <Button size="sm" variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
        )}

        {saved && (
          <span className="text-sm text-green-500 flex items-center gap-1.5">
            <CheckCircle2 className="w-3.5 h-3.5" /> Saved
          </span>
        )}

        {testResult && (
          <span
            className={cn(
              "text-sm flex items-center gap-1.5",
              testResult.ok ? "text-green-500" : "text-destructive",
            )}
          >
            {testResult.ok ? (
              <CheckCircle2 className="w-3.5 h-3.5" />
            ) : (
              <AlertCircle className="w-3.5 h-3.5" />
            )}
            {testResult.message}
          </span>
        )}
      </div>
    </div>
  );
}

export default function SettingsPage() {
  const [instances, setInstances] = useState<SonarrInstance[]>([]);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSaved, setPasswordSaved] = useState(false);

  async function handleChangePassword() {
    setPasswordError(null);
    if (newPassword !== confirmPassword) {
      setPasswordError("New passwords do not match");
      return;
    }
    if (newPassword.length < 8) {
      setPasswordError("New password must be at least 8 characters");
      return;
    }
    setPasswordSaving(true);
    try {
      await api.post("/api/auth/password", {
        current_password: currentPassword,
        new_password: newPassword,
      });
      setPasswordSaved(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setTimeout(() => setPasswordSaved(false), 3000);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setPasswordError(
          err.response?.data?.error ?? "Failed to change password",
        );
      } else {
        setPasswordError("Failed to change password");
      }
    } finally {
      setPasswordSaving(false);
    }
  }

  useEffect(() => {
    api.get("/api/settings").then((res) => setInstances(res.data.sonarr));
  }, []);

  async function addInstance(data: {
    name: string;
    url: string;
    api_key: string;
  }) {
    const res = await api.post("/api/settings/sonarr", data);
    setInstances((prev) => [...prev, res.data]);
    setShowAddForm(false);
  }

  async function updateInstance(
    id: string,
    data: { name: string; url: string; api_key: string },
  ) {
    const res = await api.put(`/api/settings/sonarr/${id}`, data);
    setInstances((prev) => prev.map((i) => (i.id === id ? res.data : i)));
    setEditingId(null);
  }

  async function deleteInstance(id: string) {
    await api.delete(`/api/settings/sonarr/${id}`);
    setInstances((prev) => prev.filter((i) => i.id !== id));
  }

  return (
    <div className="flex flex-col gap-6 max-w-2xl">
      <div>
        <h1 className="text-xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Configure your arr apps
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <KeyRound className="w-4 h-4 text-muted-foreground" />
            <CardTitle className="text-base">Change Password</CardTitle>
          </div>
          <CardDescription>Update your admin password</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label>Current Password</Label>
            <Input
              type="password"
              placeholder="Enter current password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </div>
          <div className="grid sm:grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label>New Password</Label>
              <Input
                type="password"
                placeholder="Min 8 characters"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Confirm New Password</Label>
              <Input
                type="password"
                placeholder="Repeat new password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
          </div>
          {passwordError && (
            <p className="text-sm text-destructive">{passwordError}</p>
          )}
          <div className="flex items-center gap-3">
            <Button
              size="sm"
              onClick={handleChangePassword}
              disabled={
                passwordSaving ||
                !currentPassword ||
                !newPassword ||
                !confirmPassword
              }
            >
              {passwordSaving ? (
                <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
              ) : (
                <KeyRound className="w-3.5 h-3.5 mr-1.5" />
              )}
              Update Password
            </Button>
            {passwordSaved && (
              <span className="text-sm text-green-500 flex items-center gap-1.5">
                <CheckCircle2 className="w-3.5 h-3.5" /> Password updated
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Sonarr */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Tv className="w-4 h-4 text-muted-foreground" />
              <CardTitle className="text-base">Sonarr</CardTitle>
            </div>
            {!showAddForm && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setShowAddForm(true)}
              >
                <Plus className="w-3.5 h-3.5 mr-1.5" />
                Add Instance
              </Button>
            )}
          </div>
          <CardDescription>
            Add one or more Sonarr instances (e.g. Sonarr, Sonarr 4K, Sonarr
            Anime)
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {showAddForm && (
            <div className="rounded-lg border border-border/50 p-4 bg-secondary/20">
              <p className="text-sm font-medium mb-3">New Instance</p>
              <InstanceForm
                onSave={addInstance}
                onCancel={() => setShowAddForm(false)}
                saveLabel="Add Instance"
              />
            </div>
          )}

          {instances.length === 0 && !showAddForm ? (
            <p className="text-sm text-muted-foreground">
              No instances configured yet.
            </p>
          ) : (
            instances.map((inst) => (
              <div
                key={inst.id}
                className="rounded-lg border border-border/50 p-4"
              >
                {editingId === inst.id ? (
                  <>
                    <p className="text-sm font-medium mb-3">
                      Edit — {inst.name}
                    </p>
                    <InstanceForm
                      initial={inst}
                      onSave={(data) => updateInstance(inst.id, data)}
                      onCancel={() => setEditingId(null)}
                    />
                  </>
                ) : (
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-sm font-medium">{inst.name}</p>
                      <p className="text-xs text-muted-foreground truncate mt-0.5">
                        {inst.url}
                      </p>
                      {inst.api_key_set && (
                        <p className="text-xs text-muted-foreground font-mono mt-0.5">
                          {inst.api_key_hint}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-1 shrink-0">
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7"
                        onClick={() => setEditingId(inst.id)}
                      >
                        <Pencil className="w-3.5 h-3.5" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7 text-destructive hover:text-destructive"
                        onClick={() => deleteInstance(inst.id)}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            ))
          )}
        </CardContent>
      </Card>

      {/* TODO: radarr placeholder */}
      <Card className="opacity-50">
        <CardHeader>
          <div className="flex items-center gap-2">
            <Tv className="w-4 h-4 text-muted-foreground" />
            <CardTitle className="text-base">Radarr</CardTitle>
          </div>
          <CardDescription>Coming in a future update</CardDescription>
        </CardHeader>
      </Card>
    </div>
  );
}
