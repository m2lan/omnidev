"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { UserAIConfig, TestConnectionResult, UpdateAIConfigInput } from "@/lib/api/client";
import { AIConfigForm } from "./ai-config-form";

interface AIConfigCardProps {
  config: UserAIConfig;
  onSetDefault: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onTest: (id: string) => Promise<TestConnectionResult>;
  onUpdate: (id: string, data: UpdateAIConfigInput) => Promise<void>;
}

export function AIConfigCard({ config, onSetDefault, onDelete, onTest, onUpdate }: AIConfigCardProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const handleTest = async () => {
    const result = await onTest(config.id);
    alert(result.success ? `Connection successful (${result.latency_ms}ms)` : `Connection failed: ${result.message}`);
  };

  const handleUpdate = async (data: UpdateAIConfigInput) => {
    setIsSaving(true);
    try {
      await onUpdate(config.id, data);
      setIsEditing(false);
    } catch (error) {
      console.error("Failed to update config:", error);
      alert("Failed to update configuration");
    } finally {
      setIsSaving(false);
    }
  };

  if (isEditing) {
    return (
      <AIConfigForm
        initialData={config}
        onSubmit={handleUpdate}
        onCancel={() => setIsEditing(false)}
        isLoading={isSaving}
        isEdit
      />
    );
  }

  return (
    <Card className={config.is_default ? "border-primary" : ""}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-lg flex items-center gap-2">
              {config.display_name}
              {config.is_default && (
                <span className="text-xs bg-primary text-primary-foreground px-2 py-0.5 rounded-full">
                  Default
                </span>
              )}
              {!config.is_active && (
                <span className="text-xs bg-muted text-muted-foreground px-2 py-0.5 rounded-full">
                  Inactive
                </span>
              )}
            </CardTitle>
            <CardDescription className="mt-1">
              {config.provider} · {config.protocol} · {config.models.length} models
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={handleTest}>
              Test
            </Button>
            <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
              Edit
            </Button>
            {!config.is_default && (
              <Button variant="outline" size="sm" onClick={() => onSetDefault(config.id)}>
                Set Default
              </Button>
            )}
            <Button variant="destructive" size="sm" onClick={() => onDelete(config.id)}>
              Delete
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground w-20">API Key:</span>
            <span className="font-mono">{config.api_key_mask}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground w-20">Base URL:</span>
            <span className="font-mono text-xs">{config.base_url}</span>
          </div>
          <div className="flex items-start gap-2">
            <span className="text-muted-foreground w-20">Models:</span>
            <div className="flex flex-wrap gap-1">
              {config.models.length === 0 ? (
                <span className="text-xs text-muted-foreground">All models (unrestricted)</span>
              ) : (
                config.models.map((model) => (
                  <span
                    key={model.id}
                    className="text-xs bg-muted px-2 py-0.5 rounded-full"
                  >
                    {model.display_name}
                  </span>
                ))
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
