"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface WorkflowNode {
  id: string;
  type: string;
  name: string;
  x: number;
  y: number;
  config: Record<string, unknown>;
}

interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
}

const NODE_TYPES = [
  { type: "ai", name: "AI Model", icon: "🧠", color: "bg-purple-100 text-purple-700" },
  { type: "http", name: "HTTP Request", icon: "🌐", color: "bg-blue-100 text-blue-700" },
  { type: "code", name: "Code", icon: "💻", color: "bg-green-100 text-green-700" },
  { type: "condition", name: "Condition", icon: "🔀", color: "bg-yellow-100 text-yellow-700" },
  { type: "transform", name: "Transform", icon: "🔄", color: "bg-orange-100 text-orange-700" },
  { type: "delay", name: "Delay", icon: "⏱️", color: "bg-gray-100 text-gray-700" },
];

export default function WorkflowPage() {
  const [nodes, setNodes] = useState<WorkflowNode[]>([]);
  const [edges, setEdges] = useState<WorkflowEdge[]>([]);
  const [selectedNode, setSelectedNode] = useState<WorkflowNode | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [runStatus, setRunStatus] = useState<Record<string, string>>({});

  const handleAddNode = (type: string) => {
    const nodeType = NODE_TYPES.find((n) => n.type === type);
    if (!nodeType) return;

    const newNode: WorkflowNode = {
      id: `node-${Date.now()}`,
      type,
      name: nodeType.name,
      x: 100 + nodes.length * 200,
      y: 200,
      config: {},
    };

    setNodes([...nodes, newNode]);
  };

  const handleConnect = (sourceId: string, targetId: string) => {
    const newEdge: WorkflowEdge = {
      id: `edge-${Date.now()}`,
      source: sourceId,
      target: targetId,
    };
    setEdges([...edges, newEdge]);
  };

  const handleRun = async () => {
    setIsRunning(true);
    setRunStatus({});

    for (const node of nodes) {
      setRunStatus((prev) => ({ ...prev, [node.id]: "running" }));
      await new Promise((resolve) => setTimeout(resolve, 1500));
      setRunStatus((prev) => ({ ...prev, [node.id]: "success" }));
    }

    setIsRunning(false);
  };

  const handleDeleteNode = (nodeId: string) => {
    setNodes(nodes.filter((n) => n.id !== nodeId));
    setEdges(edges.filter((e) => e.source !== nodeId && e.target !== nodeId));
    if (selectedNode?.id === nodeId) setSelectedNode(null);
  };

  return (
    <div className="flex h-full">
      {/* Node Palette */}
      <div className="w-56 border-r p-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase mb-3">Nodes</h3>
        <div className="space-y-1">
          {NODE_TYPES.map((nodeType) => (
            <button
              key={nodeType.type}
              onClick={() => handleAddNode(nodeType.type)}
              className="w-full flex items-center gap-2 rounded-lg px-3 py-2 text-sm hover:bg-muted transition-colors"
            >
              <span>{nodeType.icon}</span>
              <span>{nodeType.name}</span>
            </button>
          ))}
        </div>

        <div className="mt-6">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase mb-3">Actions</h3>
          <Button onClick={handleRun} disabled={isRunning || nodes.length === 0} className="w-full">
            {isRunning ? "Running..." : "▶ Run Workflow"}
          </Button>
        </div>
      </div>

      {/* Canvas */}
      <div className="flex-1 relative overflow-auto bg-muted/30">
        <svg className="absolute inset-0 w-full h-full" style={{ minWidth: 800, minHeight: 600 }}>
          {/* Grid */}
          <defs>
            <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
              <path d="M 20 0 L 0 0 0 20" fill="none" stroke="hsl(var(--border))" strokeWidth="0.5" />
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />

          {/* Edges */}
          {edges.map((edge) => {
            const source = nodes.find((n) => n.id === edge.source);
            const target = nodes.find((n) => n.id === edge.target);
            if (!source || !target) return null;

            return (
              <line
                key={edge.id}
                x1={source.x + 75}
                y1={source.y + 25}
                x2={target.x + 75}
                y2={target.y + 25}
                stroke="hsl(var(--primary))"
                strokeWidth="2"
                markerEnd="url(#arrowhead)"
              />
            );
          })}

          <defs>
            <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
              <polygon points="0 0, 10 3.5, 0 7" fill="hsl(var(--primary))" />
            </marker>
          </defs>
        </svg>

        {/* Nodes */}
        {nodes.map((node) => {
          const nodeType = NODE_TYPES.find((n) => n.type === node.type);
          const status = runStatus[node.id];

          return (
            <div
              key={node.id}
              onClick={() => setSelectedNode(node)}
              className={`absolute rounded-lg border bg-card shadow-sm cursor-pointer transition-all ${
                selectedNode?.id === node.id ? "ring-2 ring-primary" : ""
              } ${status === "running" ? "animate-pulse" : ""}`}
              style={{ left: node.x, top: node.y, width: 150 }}
            >
              <div className="flex items-center gap-2 p-2 border-b">
                <span>{nodeType?.icon}</span>
                <span className="text-xs font-medium truncate">{node.name}</span>
                {status && (
                  <span className={`ml-auto text-xs ${
                    status === "success" ? "text-green-600" :
                    status === "running" ? "text-blue-600" :
                    "text-red-600"
                  }`}>
                    {status === "success" ? "✓" : status === "running" ? "..." : "✗"}
                  </span>
                )}
              </div>
              <div className="p-2">
                <p className="text-xs text-muted-foreground">{nodeType?.name}</p>
              </div>
              <button
                onClick={(e) => { e.stopPropagation(); handleDeleteNode(node.id); }}
                className="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-destructive text-destructive-foreground text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 hover:opacity-100"
              >
                ×
              </button>
            </div>
          );
        })}

        {nodes.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center">
              <p className="text-4xl mb-4">⚡</p>
              <p className="text-lg font-medium mb-2">Workflow Editor</p>
              <p className="text-muted-foreground">Add nodes from the left panel to build your workflow.</p>
            </div>
          </div>
        )}
      </div>

      {/* Properties Panel */}
      {selectedNode && (
        <div className="w-72 border-l p-4">
          <h3 className="font-semibold mb-4">Node Properties</h3>
          <div className="space-y-3">
            <div>
              <label className="text-xs text-muted-foreground">Name</label>
              <p className="text-sm font-medium">{selectedNode.name}</p>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">Type</label>
              <p className="text-sm">{selectedNode.type}</p>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">ID</label>
              <p className="text-xs font-mono text-muted-foreground">{selectedNode.id}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
