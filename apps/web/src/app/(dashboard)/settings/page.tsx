"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { api, UserAIConfig, CreateAIConfigInput, UpdateAIConfigInput, TestConnectionResult, UserSettings, KnowledgeBase } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { AIConfigCard } from "@/components/settings/ai-config-card";
import { AIConfigForm } from "@/components/settings/ai-config-form";

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
  const [aiConfigs, setAiConfigs] = useState<UserAIConfig[]>([]);
  const [isLoadingConfigs, setIsLoadingConfigs] = useState(true);
  const [showConfigForm, setShowConfigForm] = useState(false);
  const [isSavingConfig, setIsSavingConfig] = useState(false);

  // RAG settings
  const [ragMode, setRagMode] = useState<"off" | "all" | "specified">("all");
  const [defaultKBIds, setDefaultKBIds] = useState<string[]>([]);
  const [availableKBs, setAvailableKBs] = useState<KnowledgeBase[]>([]);
  const [isLoadingSettings, setIsLoadingSettings] = useState(true);
  const [isSavingRAG, setIsSavingRAG] = useState(false);
  const [ragSaved, setRagSaved] = useState(false);

  // Fetch API keys, AI configs, and RAG settings on mount
  useEffect(() => {
    loadAPIKeys();
    loadAIConfigs();
    loadRAGSettings();
    loadAvailableKBs();
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

  const loadAIConfigs = async () => {
    try {
      setIsLoadingConfigs(true);
      const { data } = await api.listAIConfigs();
      setAiConfigs(data || []);
    } catch (error) {
      console.error("Failed to load AI configs:", error);
    } finally {
      setIsLoadingConfigs(false);
    }
  };

  const loadRAGSettings = async () => {
    try {
      setIsLoadingSettings(true);
      const { data } = await api.getSettings();
      if (data) {
        setRagMode((data.rag_mode as "off" | "all" | "specified") || "all");
        setDefaultKBIds((data.default_kb_ids as string[]) || []);
      }
    } catch (error) {
      console.error("Failed to load RAG settings:", error);
    } finally {
      setIsLoadingSettings(false);
    }
  };

  const loadAvailableKBs = async () => {
    try {
      const { data } = await api.listKnowledgeBases({ page_size: 100 });
      setAvailableKBs(data || []);
    } catch (error) {
      console.error("Failed to load knowledge bases:", error);
    }
  };

  const handleSaveRAGSettings = async () => {
    setIsSavingRAG(true);
    setRagSaved(false);
    try {
      await api.updateSettings({
        rag_mode: ragMode,
        default_kb_ids: ragMode === "specified" ? defaultKBIds : [],
      });
      setRagSaved(true);
      setTimeout(() => setRagSaved(false), 2000);
    } catch (error) {
      console.error("Failed to save RAG settings:", error);
    } finally {
      setIsSavingRAG(false);
    }
  };

  const toggleDefaultKB = (kbId: string) => {
    setDefaultKBIds((prev) =>
      prev.includes(kbId) ? prev.filter((id) => id !== kbId) : [...prev, kbId]
    );
  };

  const handleCreateAIConfig = async (input: CreateAIConfigInput) => {
    setIsSavingConfig(true);
    try {
      await api.createAIConfig(input);
      await loadAIConfigs();
      setShowConfigForm(false);
    } catch (error) {
      console.error("Failed to create AI config:", error);
      alert("Failed to create configuration");
    } finally {
      setIsSavingConfig(false);
    }
  };

  const handleUpdateAIConfig = async (id: string, input: UpdateAIConfigInput) => {
    try {
      await api.updateAIConfig(id, input);
      await loadAIConfigs();
    } catch (error) {
      console.error("Failed to update AI config:", error);
      throw error;
    }
  };

  const handleDeleteAIConfig = async (id: string) => {
    if (!confirm("Are you sure you want to delete this configuration?")) return;
    try {
      await api.deleteAIConfig(id);
      await loadAIConfigs();
    } catch (error) {
      console.error("Failed to delete AI config:", error);
    }
  };

  const handleSetDefaultAIConfig = async (id: string) => {
    try {
      await api.setDefaultAIConfig(id);
      await loadAIConfigs();
    } catch (error) {
      console.error("Failed to set default AI config:", error);
    }
  };

  const handleTestAIConfig = async (id: string): Promise<TestConnectionResult> => {
    const { data } = await api.testAIConfigConnection(id);
    return data;
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

      {/* AI Configs */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>AI Providers</CardTitle>
              <CardDescription>Configure your AI provider connections</CardDescription>
            </div>
            {!showConfigForm && (
              <Button onClick={() => setShowConfigForm(true)}>Add Provider</Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {showConfigForm && (
            <div className="mb-6">
              <AIConfigForm
                onSubmit={handleCreateAIConfig}
                onCancel={() => setShowConfigForm(false)}
                isLoading={isSavingConfig}
              />
            </div>
          )}

          {isLoadingConfigs ? (
            <p className="text-sm text-muted-foreground">Loading configurations...</p>
          ) : aiConfigs.length === 0 ? (
            <p className="text-sm text-muted-foreground">No AI providers configured yet. Add one above.</p>
          ) : (
            <div className="space-y-4">
              {aiConfigs.map((config) => (
                <AIConfigCard
                  key={config.id}
                  config={config}
                  onSetDefault={handleSetDefaultAIConfig}
                  onDelete={handleDeleteAIConfig}
                  onTest={handleTestAIConfig}
                  onUpdate={handleUpdateAIConfig}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* RAG Knowledge Settings */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>📚 RAG Knowledge Settings</CardTitle>
          <CardDescription>Configure how knowledge bases are used in conversations</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoadingSettings ? (
            <p className="text-sm text-muted-foreground">Loading settings...</p>
          ) : (
            <>
              <div className="space-y-3">
                <Label>RAG Mode</Label>
                <div className="space-y-2">
                  {[
                    { value: "off", label: "Off", desc: "Don't search knowledge bases" },
                    { value: "all", label: "All (Recommended)", desc: "Automatically search all your knowledge bases" },
                    { value: "specified", label: "Specified", desc: "Only use selected knowledge bases below" },
                  ].map((option) => (
                    <div
                      key={option.value}
                      className={`flex items-center gap-3 p-3 border rounded-lg cursor-pointer transition-colors ${
                        ragMode === option.value ? "border-primary bg-primary/5" : "hover:bg-muted/50"
                      }`}
                      onClick={() => setRagMode(option.value as "off" | "all" | "specified")}
                    >
                      <div
                        className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                          ragMode === option.value ? "border-primary" : "border-muted-foreground"
                        }`}
                      >
                        {ragMode === option.value && (
                          <div className="w-2 h-2 rounded-full bg-primary" />
                        )}
                      </div>
                      <div>
                        <p className="text-sm font-medium">{option.label}</p>
                        <p className="text-xs text-muted-foreground">{option.desc}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {ragMode === "specified" && (
                <div className="space-y-3">
                  <Label>Default Knowledge Bases</Label>
                  <p className="text-xs text-muted-foreground">
                    Select which knowledge bases to use by default in new conversations
                  </p>
                  {availableKBs.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No knowledge bases available. Create one first.</p>
                  ) : (
                    <div className="space-y-2 max-h-48 overflow-y-auto">
                      {availableKBs.map((kb) => (
                        <div
                          key={kb.id}
                          className={`flex items-center gap-3 p-3 border rounded-lg cursor-pointer transition-colors ${
                            defaultKBIds.includes(kb.id) ? "border-primary bg-primary/5" : "hover:bg-muted/50"
                          }`}
                          onClick={() => toggleDefaultKB(kb.id)}
                        >
                          <div
                            className={`w-4 h-4 rounded border flex items-center justify-center ${
                              defaultKBIds.includes(kb.id)
                                ? "bg-primary border-primary"
                                : "border-muted-foreground"
                            }`}
                          >
                            {defaultKBIds.includes(kb.id) && (
                              <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                              </svg>
                            )}
                          </div>
                          <div className="flex-1">
                            <p className="text-sm font-medium">{kb.name}</p>
                            <p className="text-xs text-muted-foreground">
                              {kb.doc_count} docs · {kb.chunk_count} chunks
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="flex items-center gap-2">
                <Button onClick={handleSaveRAGSettings} disabled={isSavingRAG}>
                  {isSavingRAG ? "Saving..." : ragSaved ? "Saved!" : "Save RAG Settings"}
                </Button>
              </div>
            </>
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
