"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { UserAIConfig, TestConnectionResult } from "@/lib/api/client";

interface AIConfigCardProps {
  config: UserAIConfig;
  onSetDefault: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onTest: (id: string) => Promise<TestConnectionResult>;
}

export function AIConfigCard({ config, onSetDefault, onDelete, onTest }: AIConfigCardProps) {
  const handleTest = async () => {
    const result = await onTest(config.id);
    alert(result.success ? `Connection successful (${result.latency_ms}ms)` : `Connection failed: ${result.message}`);
  };

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
              {config.models.map((model) => (
                <span
                  key={model.id}
                  className="text-xs bg-muted px-2 py-0.5 rounded-full"
                >
                  {model.display_name}
                </span>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
