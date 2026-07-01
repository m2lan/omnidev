"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  created_at: string;
  expires_at?: string;
}

export default function SettingsPage() {
  const router = useRouter();
  const { user, logout, fetchProfile } = useAuthStore();
  const [nickname, setNickname] = useState(user?.nickname || "");
  const [bio, setBio] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [isLoadingKeys, setIsLoadingKeys] = useState(true);
  const [newKeyName, setNewKeyName] = useState("");
  const [isCreatingKey, setIsCreatingKey] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  // Fetch API keys on mount
  useEffect(() => {
    loadAPIKeys();
  }, []);

  // Update local state when user changes
  useEffect(() => {
    if (user) {
      setNickname(user.nickname || "");
    }
  }, [user]);

  const loadAPIKeys = async () => {
    try {
      setIsLoadingKeys(true);
      const { data } = await api.listAPIKeys();
      setApiKeys(data || []);
    } catch (error) {
      console.error("Failed to load API keys:", error);
    } finally {
      setIsLoadingKeys(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.updateProfile({ nickname, bio });
      await fetchProfile();
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (error) {
      console.error("Failed to save profile:", error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleCreateKey = async () => {
    if (!newKeyName.trim()) return;
    setIsCreatingKey(true);
    try {
      const { data } = await api.createAPIKey({ name: newKeyName });
      setCreatedKey(data.key);
      setNewKeyName("");
      await loadAPIKeys();
    } catch (error) {
      console.error("Failed to create API key:", error);
    } finally {
      setIsCreatingKey(false);
    }
  };

  const handleRevokeKey = async (keyId: string) => {
    try {
      await api.revokeAPIKey(keyId);
      await loadAPIKeys();
    } catch (error) {
      console.error("Failed to revoke API key:", error);
    }
  };

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Settings</h1>

      {/* Profile */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Manage your personal information</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <Avatar className="h-16 w-16">
              <AvatarFallback className="text-lg">
                {user?.nickname?.[0]?.toUpperCase() || "U"}
              </AvatarFallback>
            </Avatar>
            <div>
              <p className="font-medium">{user?.nickname}</p>
              <p className="text-sm text-muted-foreground">{user?.email}</p>
              <p className="text-xs text-muted-foreground capitalize">Role: {user?.role}</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="nickname">Nickname</Label>
            <Input id="nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" value={user?.email || ""} disabled />
          </div>

          <div className="space-y-2">
            <Label htmlFor="bio">Bio</Label>
            <Input id="bio" value={bio} onChange={(e) => setBio(e.target.value)} placeholder="Tell us about yourself" />
          </div>

          <div className="flex items-center gap-2">
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Saving..." : saved ? "Saved!" : "Save Changes"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* API Keys */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>API Keys</CardTitle>
          <CardDescription>Manage your API keys for programmatic access</CardDescription>
        </CardHeader>
        <CardContent>
          {createdKey && (
            <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg">
              <p className="text-sm font-medium text-green-800">API Key Created!</p>
              <p className="text-xs text-green-600 font-mono mt-1 break-all">{createdKey}</p>
              <p className="text-xs text-green-600 mt-1">Copy this key now - it won't be shown again.</p>
              <Button variant="outline" size="sm" className="mt-2" onClick={() => setCreatedKey(null)}>
                Dismiss
              </Button>
            </div>
          )}

          <div className="flex gap-2 mb-4">
            <Input
              placeholder="Key name (e.g., Production)"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
            />
            <Button onClick={handleCreateKey} disabled={isCreatingKey || !newKeyName.trim()}>
              {isCreatingKey ? "Creating..." : "Create Key"}
            </Button>
          </div>

          {isLoadingKeys ? (
            <p className="text-sm text-muted-foreground">Loading keys...</p>
          ) : apiKeys.length === 0 ? (
            <p className="text-sm text-muted-foreground">No API keys yet. Create one above.</p>
          ) : (
            <div className="space-y-2">
              {apiKeys.map((key) => (
                <div key={key.id} className="flex items-center justify-between p-3 border rounded-lg">
                  <div>
                    <p className="text-sm font-medium">{key.name}</p>
                    <p className="text-xs text-muted-foreground font-mono">{key.key_prefix}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => handleRevokeKey(key.id)}>
                    Revoke
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Billing */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Billing</CardTitle>
          <CardDescription>Manage your subscription and billing</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between p-4 border rounded-lg">
            <div>
              <p className="font-medium">Free Plan</p>
              <p className="text-sm text-muted-foreground">$10 credit remaining</p>
            </div>
            <Button>Upgrade to Pro</Button>
          </div>
        </CardContent>
      </Card>

      {/* Notifications */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Notifications</CardTitle>
          <CardDescription>Configure notification preferences</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {[
            { label: "Email notifications", desc: "Receive email for important updates" },
            { label: "Agent completion", desc: "Notify when agent tasks complete" },
            { label: "Deployment status", desc: "Notify on deploy success/failure" },
            { label: "Billing alerts", desc: "Notify when approaching budget limit" },
          ].map((item) => (
            <div key={item.label} className="flex items-center justify-between p-3 border rounded-lg">
              <div>
                <p className="text-sm font-medium">{item.label}</p>
                <p className="text-xs text-muted-foreground">{item.desc}</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" className="sr-only peer" defaultChecked />
                <div className="w-9 h-5 bg-muted peer-focus:ring-2 peer-focus:ring-ring rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary" />
              </label>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Sign out</p>
              <p className="text-xs text-muted-foreground">Sign out from this device</p>
            </div>
            <Button variant="outline" onClick={handleLogout}>Sign Out</Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Delete account</p>
              <p className="text-xs text-muted-foreground">Permanently delete your account and all data</p>
            </div>
            <Button variant="destructive" disabled>Delete Account</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
