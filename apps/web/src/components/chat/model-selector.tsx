"use client";

import { useState, useEffect } from "react";
import { api, type Model } from "@/lib/api/client";

interface ModelSelectorProps {
  value: string;
  onChange: (model: string) => void;
}

export function ModelSelector({ value, onChange }: ModelSelectorProps) {
  const [models, setModels] = useState<Model[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadModels();
  }, []);

  const loadModels = async () => {
    try {
      const { data } = await api.listModels();
      setModels(data || []);
      // Auto-select first model if none selected
      if (!value && data && data.length > 0) {
        onChange(data[0].model_id);
      }
    } catch {
      // If models API fails, use hardcoded defaults
      const defaults = [
        { id: "1", provider: "deepseek", model_id: "deepseek-chat", display_name: "DeepSeek Chat", context_window: 32768, supports_streaming: true, supports_vision: false, supports_tools: false },
        { id: "2", provider: "openai", model_id: "gpt-4o-mini", display_name: "GPT-4o Mini", context_window: 128000, supports_streaming: true, supports_vision: true, supports_tools: true },
        { id: "3", provider: "openai", model_id: "mimo-v2.5-pro", display_name: "MiMo v2.5 Pro", context_window: 32768, supports_streaming: true, supports_vision: false, supports_tools: false },
      ];
      setModels(defaults);
      if (!value) {
        onChange(defaults[0].model_id);
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span className="animate-pulse">Loading models...</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="model-select" className="text-xs text-muted-foreground">
        Model:
      </label>
      <select
        id="model-select"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-md border bg-background px-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-ring"
      >
        {models.map((model) => (
          <option key={model.id} value={model.model_id}>
            {model.display_name || model.model_id}
          </option>
        ))}
      </select>
    </div>
  );
}
