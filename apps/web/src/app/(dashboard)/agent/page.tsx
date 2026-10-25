"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ACTIVE_RUN_STATUSES,
  Agent,
  AgentRun,
  AgentRunDetail,
  api,
} from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const AVAILABLE_TOOLS = [
  { name: "file", label: "Workspace files" },
  { name: "search", label: "Workspace search" },
  { name: "calculator", label: "Calculator" },
  { name: "code_exec", label: "Code execution (Docker sandbox)" },
];

function isRunActive(status: string) {
  return ACTIVE_RUN_STATUSES.includes(status as AgentRun["status"]);
}

export default function AgentPage() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");
  const [maxSteps, setMaxSteps] = useState(12);
  const [enabledTools, setEnabledTools] = useState<string[]>(["file", "code_exec"]);
  const [isSaving, setIsSaving] = useState(false);

  const [task, setTask] = useState("");
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [detail, setDetail] = useState<AgentRunDetail | null>(null);
  const [isStarting, setIsStarting] = useState(false);

  const selectedAgent = agents.find((agent) => agent.id === selectedId) || null;

  const loadAgents = useCallback(async () => {
    try {
      const { data } = await api.listAgents({ page_size: 50 });
      setAgents(data || []);
      setLoadError(null);
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "Failed to load agents");
    }
  }, []);

  const loadRuns = useCallback(async (agentId: string) => {
    try {
      const { data } = await api.listAgentRuns(agentId, { page_size: 20 });
      setRuns(data || []);
    } catch {
      setRuns([]);
    }
  }, []);

  const loadRun = useCallback(async (runId: string) => {
    try {
      const { data } = await api.getAgentRun(runId);
      setDetail(data);
      setActionError(null);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to load run");
    }
  }, []);

  useEffect(() => {
    loadAgents();
  }, [loadAgents]);

  useEffect(() => {
    setDetail(null);
    setRuns([]);
    if (selectedId) {
      loadRuns(selectedId);
    }
  }, [selectedId, loadRuns]);

  // Poll the open run until it reaches a terminal state.
  useEffect(() => {
    if (!detail || !isRunActive(detail.run.status)) return;
    const runId = detail.run.id;
    const timer = setTimeout(async () => {
      await loadRun(runId);
      if (selectedId) {
        loadRuns(selectedId);
      }
    }, 2000);
    return () => clearTimeout(timer);
  }, [detail, loadRun, loadRuns, selectedId]);

  const handleCreate = async () => {
    if (!name.trim() || !systemPrompt.trim()) return;
    setIsSaving(true);
    setActionError(null);
    try {
      const { data } = await api.createAgent({
        name: name.trim(),
        description: description.trim(),
        system_prompt: systemPrompt.trim(),
        tools: enabledTools.map((toolName) => ({ name: toolName, enabled: true })),
        config: { max_steps: maxSteps },
      });
      setShowCreate(false);
      setName("");
      setDescription("");
      setSystemPrompt("");
      await loadAgents();
      setSelectedId(data.id);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to create agent");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (agent: Agent) => {
    if (!confirm(`Delete agent "${agent.name}"?`)) return;
    try {
      await api.deleteAgent(agent.id);
      if (selectedId === agent.id) {
        setSelectedId(null);
      }
      await loadAgents();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to delete agent");
    }
  };

  const handleRun = async () => {
    if (!selectedId || !task.trim()) return;
    setIsStarting(true);
    setActionError(null);
    try {
      const { data } = await api.runAgent(selectedId, task.trim());
      setDetail({ run: data, steps: [] });
      setTask("");
      await loadRuns(selectedId);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to start run");
    } finally {
      setIsStarting(false);
    }
  };

  const handleCancel = async () => {
    if (!detail) return;
    try {
      await api.cancelAgentRun(detail.run.id);
      await loadRun(detail.run.id);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Failed to cancel run");
    }
  };

  return (
    <div className="flex h-full">
      <div className="w-80 border-r p-4 space-y-4 overflow-auto">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Agents</h2>
          <Button size="sm" variant="outline" onClick={() => setShowCreate((v) => !v)}>
            {showCreate ? "Close" : "New"}
          </Button>
        </div>

        {showCreate && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">New agent</CardTitle>
              <CardDescription>Runs asynchronously in a Docker sandbox</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="agent-name">Name</Label>
                <Input id="agent-name" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label htmlFor="agent-description">Description</Label>
                <Input
                  id="agent-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="agent-prompt">System prompt</Label>
                <textarea
                  id="agent-prompt"
                  className="w-full min-h-[100px] rounded-md border border-input bg-background px-3 py-2 text-sm"
                  value={systemPrompt}
                  onChange={(e) => setSystemPrompt(e.target.value)}
                  placeholder="You are a coding assistant. Solve the task step by step."
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="agent-steps">Max steps</Label>
                <Input
                  id="agent-steps"
                  type="number"
                  min={1}
                  max={50}
                  value={maxSteps}
                  onChange={(e) => setMaxSteps(Number(e.target.value) || 1)}
                />
              </div>
              <div className="space-y-1">
                <Label>Tools</Label>
                {AVAILABLE_TOOLS.map((tool) => (
                  <label key={tool.name} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={enabledTools.includes(tool.name)}
                      onChange={(e) =>
                        setEnabledTools((prev) =>
                          e.target.checked
                            ? [...prev, tool.name]
                            : prev.filter((t) => t !== tool.name)
                        )
                      }
                    />
                    {tool.label}
                  </label>
                ))}
              </div>
              <Button
                className="w-full"
                onClick={handleCreate}
                disabled={isSaving || !name.trim() || !systemPrompt.trim()}
              >
                {isSaving ? "Creating..." : "Create agent"}
              </Button>
            </CardContent>
          </Card>
        )}

        {loadError && <p className="text-sm text-destructive">{loadError}</p>}
        {!loadError && agents.length === 0 && (
          <p className="text-sm text-muted-foreground">
            No agents yet. Create one to run background tasks.
          </p>
        )}

        <div className="space-y-2">
          {agents.map((agent) => (
            <div
              key={agent.id}
              onClick={() => setSelectedId(agent.id)}
              className={`rounded-lg border p-3 cursor-pointer transition-colors ${
                selectedId === agent.id ? "bg-accent border-primary" : "hover:bg-muted"
              }`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium">{agent.name}</span>
                <button
                  className="text-xs text-muted-foreground hover:text-destructive"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(agent);
                  }}
                >
                  Delete
                </button>
              </div>
              <p className="text-xs text-muted-foreground mt-1">{agent.description}</p>
              <div className="flex flex-wrap gap-1 mt-2">
                {(agent.tools || []).map((tool) => (
                  <span key={tool.name} className="text-xs bg-muted px-1.5 py-0.5 rounded">
                    {tool.name}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="flex-1 flex flex-col overflow-hidden">
        {selectedAgent ? (
          <>
            <div className="border-b p-4">
              <h2 className="text-lg font-semibold">{selectedAgent.name}</h2>
              <p className="text-sm text-muted-foreground line-clamp-2">
                {selectedAgent.system_prompt}
              </p>
            </div>

            <div className="border-b p-4 space-y-2">
              <div className="flex gap-2">
                <Input
                  value={task}
                  onChange={(e) => setTask(e.target.value)}
                  placeholder="Describe a task, e.g. write a Python script that computes fibonacci numbers"
                  onKeyDown={(e) => e.key === "Enter" && handleRun()}
                  disabled={isStarting}
                />
                <Button onClick={handleRun} disabled={isStarting || !task.trim()}>
                  {isStarting ? "Starting..." : "Run"}
                </Button>
              </div>
              {actionError && <p className="text-sm text-destructive">{actionError}</p>}
              {runs.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {runs.map((run) => (
                    <button
                      key={run.id}
                      onClick={() => loadRun(run.id)}
                      className={`text-xs rounded border px-2 py-1 ${
                        detail?.run.id === run.id ? "border-primary bg-accent" : "hover:bg-muted"
                      }`}
                    >
                      {new Date(run.created_at).toLocaleString()} · {run.status}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div className="flex-1 overflow-auto p-4">
              {detail ? (
                <div className="max-w-3xl mx-auto space-y-4">
                  <div className="rounded-lg bg-muted p-4">
                    <div className="flex items-center justify-between gap-2 mb-1">
                      <p className="text-sm font-medium">
                        {detail.run.status}
                        {isRunActive(detail.run.status) &&
                          ` · ${detail.run.completed_steps}/${detail.run.total_steps} steps`}
                      </p>
                      {isRunActive(detail.run.status) && (
                        <Button size="sm" variant="outline" onClick={handleCancel}>
                          Cancel
                        </Button>
                      )}
                    </div>
                    <p className="text-sm whitespace-pre-wrap">{detail.run.task}</p>
                  </div>

                  {detail.run.status !== "created" && detail.steps.length === 0 && (
                    <p className="text-sm text-muted-foreground">Waiting for the first step...</p>
                  )}

                  {detail.steps.map((step) => (
                    <div key={step.id} className="rounded-lg border p-3 space-y-2">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-medium bg-muted px-2 py-0.5 rounded">
                          #{step.step_number} {step.type}
                          {step.tool_name ? ` · ${step.tool_name}` : ""}
                        </span>
                        <span
                          className={`text-xs ${
                            step.status === "failed" ? "text-destructive" : "text-muted-foreground"
                          }`}
                        >
                          {step.status}
                        </span>
                      </div>
                      {step.content && (
                        <p className="text-sm whitespace-pre-wrap">{step.content}</p>
                      )}
                      {step.error && <p className="text-sm text-destructive">{step.error}</p>}
                      {step.tool_output && (
                        <pre className="text-xs bg-muted rounded p-2 overflow-auto max-h-64">
                          {JSON.stringify(step.tool_output, null, 2)}
                        </pre>
                      )}
                    </div>
                  ))}

                  {detail.run.result && (
                    <div className="rounded-lg border p-4">
                      <p className="text-sm font-medium mb-1">Result</p>
                      <p className="text-sm whitespace-pre-wrap">{detail.run.result}</p>
                    </div>
                  )}
                  {detail.run.error && (
                    <div className="rounded-lg border border-destructive p-4">
                      <p className="text-sm font-medium mb-1">Error</p>
                      <p className="text-sm whitespace-pre-wrap">{detail.run.error}</p>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <p className="text-lg font-medium mb-2">Background agent runs</p>
                  <p className="text-muted-foreground max-w-md">
                    Describe a task and run it. Steps execute asynchronously; code runs inside an
                    isolated Docker container with no network access.
                  </p>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <p className="text-lg font-medium mb-2">Select an agent</p>
              <p className="text-muted-foreground">Choose an agent or create a new one.</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
