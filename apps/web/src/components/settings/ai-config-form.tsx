"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CreateAIConfigInput, ModelConfig } from "@/lib/api/client";

interface AIConfigFormProps {
  onSubmit: (data: CreateAIConfigInput) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

const PROVIDERS = [
  { value: "openai", label: "OpenAI", defaultUrl: "https://api.openai.com/v1" },
  { value: "anthropic", label: "Anthropic", defaultUrl: "https://api.anthropic.com" },
  { value: "deepseek", label: "DeepSeek", defaultUrl: "https://api.deepseek.com/v1" },
  { value: "custom", label: "Custom (OpenAI-compatible)", defaultUrl: "" },
];

const DEFAULT_MODELS: Record<string, ModelConfig[]> = {
  openai: [
    { id: "gpt-4o", display_name: "GPT-4o" },
    { id: "gpt-4o-mini", display_name: "GPT-4o Mini" },
    { id: "gpt-4-turbo", display_name: "GPT-4 Turbo" },
  ],
  anthropic: [
    { id: "claude-opus-4-8", display_name: "Claude Opus 4.8" },
    { id: "claude-sonnet-4-6", display_name: "Claude Sonnet 4.6" },
    { id: "claude-haiku-4-5-20251001", display_name: "Claude Haiku 4.5" },
  ],
  deepseek: [
    { id: "deepseek-chat", display_name: "DeepSeek Chat" },
    { id: "deepseek-coder", display_name: "DeepSeek Coder" },
    { id: "deepseek-reasoner", display_name: "DeepSeek Reasoner" },
  ],
  custom: [],
};

export function AIConfigForm({ onSubmit, onCancel, isLoading }: AIConfigFormProps) {
  const [provider, setProvider] = useState("openai");
  const [displayName, setDisplayName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [baseUrl, setBaseUrl] = useState("https://api.openai.com/v1");
  const [protocol, setProtocol] = useState<"openai" | "anthropic">("openai");
  const [models, setModels] = useState<ModelConfig[]>(DEFAULT_MODELS.openai);
  const [isDefault, setIsDefault] = useState(false);
  const [newModelId, setNewModelId] = useState("");
  const [newModelName, setNewModelName] = useState("");

  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider);
    const providerInfo = PROVIDERS.find((p) => p.value === newProvider);
    if (providerInfo) {
      setBaseUrl(providerInfo.defaultUrl);
    }
    if (newProvider === "anthropic") {
      setProtocol("anthropic");
    } else {
      setProtocol("openai");
    }
    setModels(DEFAULT_MODELS[newProvider] || []);
  };

  const handleAddModel = () => {
    if (!newModelId.trim()) return;
    const newModel: ModelConfig = {
      id: newModelId.trim(),
      display_name: newModelName.trim() || newModelId.trim(),
    };
    setModels([...models, newModel]);
    setNewModelId("");
    setNewModelName("");
  };

  const handleRemoveModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      provider,
      display_name: displayName,
      api_key: apiKey,
      base_url: baseUrl,
      protocol,
      models,
      is_default: isDefault,
    };
    console.log("Submitting AI config:", data);
    await onSubmit(data);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Add AI Provider</CardTitle>
        <CardDescription>Configure a new AI provider connection</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Provider */}
          <div className="space-y-2">
            <Label htmlFor="provider">Provider</Label>
            <select
              id="provider"
              value={provider}
              onChange={(e) => handleProviderChange(e.target.value)}
              className="w-full p-2 border rounded-md bg-background"
            >
              {PROVIDERS.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </div>

          {/* Display Name */}
          <div className="space-y-2">
            <Label htmlFor="display_name">Display Name</Label>
            <Input
              id="display_name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="e.g., My OpenAI Account"
              required
            />
          </div>

          {/* API Key */}
          <div className="space-y-2">
            <Label htmlFor="api_key">API Key</Label>
            <Input
              id="api_key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-..."
              required
            />
          </div>

          {/* Base URL */}
          <div className="space-y-2">
            <Label htmlFor="base_url">Base URL</Label>
            <Input
              id="base_url"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="https://api.openai.com/v1"
              required
            />
          </div>

          {/* Protocol */}
          <div className="space-y-2">
            <Label htmlFor="protocol">Protocol</Label>
            <select
              id="protocol"
              value={protocol}
              onChange={(e) => setProtocol(e.target.value as "openai" | "anthropic")}
              className="w-full p-2 border rounded-md bg-background"
            >
              <option value="openai">OpenAI-compatible</option>
              <option value="anthropic">Anthropic</option>
            </select>
          </div>

          {/* Models */}
          <div className="space-y-2">
            <Label>Models</Label>
            <div className="space-y-2">
              {models.map((model, index) => (
                <div key={index} className="flex items-center gap-2 p-2 border rounded-md">
                  <div className="flex-1">
                    <p className="text-sm font-medium">{model.display_name}</p>
                    <p className="text-xs text-muted-foreground font-mono">{model.id}</p>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => handleRemoveModel(index)}
                  >
                    Remove
                  </Button>
                </div>
              ))}
            </div>

            <div className="flex gap-2">
              <Input
                placeholder="Model ID (e.g., gpt-4o)"
                value={newModelId}
                onChange={(e) => setNewModelId(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleAddModel();
                  }
                }}
              />
              <Input
                placeholder="Display name"
                value={newModelName}
                onChange={(e) => setNewModelName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleAddModel();
                  }
                }}
              />
              <Button type="button" variant="outline" onClick={handleAddModel}>
                Add
              </Button>
            </div>
          </div>

          {/* Default */}
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is_default"
              checked={isDefault}
              onChange={(e) => setIsDefault(e.target.checked)}
              className="rounded"
            />
            <Label htmlFor="is_default">Set as default provider</Label>
          </div>

          {/* Actions */}
          <div className="flex gap-2">
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Saving..." : "Save Configuration"}
            </Button>
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
