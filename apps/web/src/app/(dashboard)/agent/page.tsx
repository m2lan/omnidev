"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface Agent {
  id: string;
  name: string;
  description: string;
  system_prompt: string;
  tools: string[];
  status: string;
}

interface AgentRun {
  id: string;
  task: string;
  status: string;
  result: string;
  steps: { step: number; type: string; content: string; status: string }[];
}

export default function AgentPage() {
  const [agents] = useState<Agent[]>([
    { id: "1", name: "Code Assistant", description: "Helps write and debug code", system_prompt: "You are a coding assistant.", tools: ["file", "code_exec"], status: "active" },
    { id: "2", name: "Research Agent", description: "Searches and analyzes information", system_prompt: "You are a research assistant.", tools: ["search", "browser"], status: "active" },
    { id: "3", name: "Data Analyst", description: "Analyzes data and creates reports", system_prompt: "You are a data analyst.", tools: ["code_exec", "calculator"], status: "active" },
  ]);

  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [task, setTask] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [currentRun, setCurrentRun] = useState<AgentRun | null>(null);

  const handleRun = async () => {
    if (!selectedAgent || !task.trim()) return;

    setIsRunning(true);
    setCurrentRun({
      id: "run-1",
      task,
      status: "running",
      result: "",
      steps: [],
    });

    // Simulate execution
    const steps = [
      { step: 1, type: "plan", content: "Analyzing task and creating plan...", status: "success" },
      { step: 2, type: "think", content: "Breaking down the task into subtasks", status: "success" },
      { step: 3, type: "tool_call", content: "Using file tool to read project structure", status: "success" },
      { step: 4, type: "code_exec", content: "Executing generated code", status: "success" },
      { step: 5, type: "response", content: "Task completed successfully!", status: "success" },
    ];

    for (let i = 0; i < steps.length; i++) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setCurrentRun((prev) => prev ? {
        ...prev,
        steps: [...prev.steps, steps[i]],
        status: i === steps.length - 1 ? "success" : "running",
        result: i === steps.length - 1 ? "Agent completed the task successfully." : "",
      } : null);
    }

    setIsRunning(false);
  };

  return (
    <div className="flex h-full">
      {/* Agent List */}
      <div className="w-80 border-r p-4">
        <h2 className="text-lg font-semibold mb-4">Agents</h2>
        <div className="space-y-2">
          {agents.map((agent) => (
            <div
              key={agent.id}
              onClick={() => setSelectedAgent(agent)}
              className={`rounded-lg border p-3 cursor-pointer transition-colors ${
                selectedAgent?.id === agent.id ? "bg-accent border-primary" : "hover:bg-muted"
              }`}
            >
              <div className="flex items-center gap-2 mb-1">
                <span>🤖</span>
                <span className="font-medium">{agent.name}</span>
              </div>
              <p className="text-xs text-muted-foreground">{agent.description}</p>
              <div className="flex gap-1 mt-2">
                {agent.tools.map((tool) => (
                  <span key={tool} className="text-xs bg-muted px-1.5 py-0.5 rounded">{tool}</span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Agent Playground */}
      <div className="flex-1 flex flex-col">
        {selectedAgent ? (
          <>
            {/* Header */}
            <div className="border-b p-4">
              <h2 className="text-lg font-semibold">{selectedAgent.name}</h2>
              <p className="text-sm text-muted-foreground">{selectedAgent.system_prompt}</p>
            </div>

            {/* Task Input */}
            <div className="border-b p-4">
              <div className="flex gap-2">
                <Input
                  value={task}
                  onChange={(e) => setTask(e.target.value)}
                  placeholder="Describe your task..."
                  onKeyDown={(e) => e.key === "Enter" && handleRun()}
                  disabled={isRunning}
                />
                <Button onClick={handleRun} disabled={isRunning || !task.trim()}>
                  {isRunning ? "Running..." : "Run"}
                </Button>
              </div>
            </div>

            {/* Execution View */}
            <div className="flex-1 overflow-auto p-4">
              {currentRun ? (
                <div className="max-w-3xl mx-auto space-y-4">
                  {/* Task */}
                  <div className="rounded-lg bg-muted p-4">
                    <p className="text-sm font-medium mb-1">Task</p>
                    <p className="text-sm">{currentRun.task}</p>
                  </div>

                  {/* Steps */}
                  {currentRun.steps.map((step) => (
                    <div key={step.step} className="flex gap-3 animate-fade-in">
                      <div className="flex flex-col items-center">
                        <div className={`h-8 w-8 rounded-full flex items-center justify-center text-xs font-medium ${
                          step.status === "success" ? "bg-green-100 text-green-700" :
                          step.status === "failed" ? "bg-red-100 text-red-700" :
                          "bg-blue-100 text-blue-700"
                        }`}>
                          {step.step}
                        </div>
                        {step.step < (currentRun.steps.length) && (
                          <div className="w-px h-full bg-border" />
                        )}
                      </div>
                      <div className="flex-1 pb-4">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-xs font-medium bg-muted px-2 py-0.5 rounded">{step.type}</span>
                          <span className={`text-xs ${step.status === "success" ? "text-green-600" : "text-red-600"}`}>
                            {step.status}
                          </span>
                        </div>
                        <p className="text-sm">{step.content}</p>
                      </div>
                    </div>
                  ))}

                  {/* Result */}
                  {currentRun.result && (
                    <div className="rounded-lg border border-green-200 bg-green-50 p-4">
                      <p className="text-sm font-medium text-green-700 mb-1">Result</p>
                      <p className="text-sm">{currentRun.result}</p>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center">
                  <p className="text-4xl mb-4">🤖</p>
                  <p className="text-lg font-medium mb-2">Agent Playground</p>
                  <p className="text-muted-foreground">Select an agent and describe a task to run.</p>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <p className="text-4xl mb-4">🤖</p>
              <p className="text-lg font-medium mb-2">Select an Agent</p>
              <p className="text-muted-foreground">Choose an agent from the list to get started.</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
