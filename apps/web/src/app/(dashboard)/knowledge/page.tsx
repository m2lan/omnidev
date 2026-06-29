"use client";

import { useEffect, useState } from "react";
import { api, type KnowledgeBase } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatRelativeTime } from "@/lib/utils";

export default function KnowledgePage() {
  const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBase[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<unknown[]>([]);
  const [selectedKB, setSelectedKB] = useState<string | null>(null);

  useEffect(() => {
    loadKnowledgeBases();
  }, []);

  const loadKnowledgeBases = async () => {
    try {
      const { data } = await api.listKnowledgeBases();
      setKnowledgeBases(data || []);
    } catch (err) {
      console.error("Failed to load knowledge bases:", err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = async () => {
    if (!newName.trim()) return;
    try {
      await api.createKnowledgeBase({ name: newName, description: newDesc });
      setNewName("");
      setNewDesc("");
      setShowCreate(false);
      loadKnowledgeBases();
    } catch (err) {
      console.error("Failed to create knowledge base:", err);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim() || !selectedKB) return;
    try {
      const { data } = await api.searchKnowledgeBase(selectedKB, searchQuery);
      setSearchResults(data || []);
    } catch (err) {
      console.error("Search failed:", err);
    }
  };

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Knowledge Base</h1>
          <p className="text-muted-foreground">Upload documents and search across your knowledge.</p>
        </div>
        <Button onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? "Cancel" : "+ Create"}
        </Button>
      </div>

      {/* Create Form */}
      {showCreate && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create Knowledge Base</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Name</Label>
              <Input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="My Knowledge Base" />
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Input value={newDesc} onChange={(e) => setNewDesc(e.target.value)} placeholder="Optional description" />
            </div>
            <Button onClick={handleCreate}>Create</Button>
          </CardContent>
        </Card>
      )}

      {/* Search */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Search</CardTitle>
          <CardDescription>Search across your knowledge bases</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2">
            <select
              className="rounded-md border bg-background px-3 py-2 text-sm"
              value={selectedKB || ""}
              onChange={(e) => setSelectedKB(e.target.value)}
            >
              <option value="">Select knowledge base</option>
              {knowledgeBases.map((kb) => (
                <option key={kb.id} value={kb.id}>{kb.name}</option>
              ))}
            </select>
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search query..."
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="flex-1"
            />
            <Button onClick={handleSearch}>Search</Button>
          </div>

          {searchResults.length > 0 && (
            <div className="mt-4 space-y-2">
              {searchResults.map((result: any, i) => (
                <div key={i} className="rounded-lg border p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-muted-foreground">Score: {result.score?.toFixed(3)}</span>
                    <span className="text-xs bg-muted px-2 py-0.5 rounded">{result.source}</span>
                  </div>
                  <p className="text-sm">{result.chunk?.content?.substring(0, 200)}...</p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Knowledge Base List */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="h-40" />
            </Card>
          ))}
        </div>
      ) : knowledgeBases.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <p className="text-4xl mb-4">📚</p>
            <p className="text-lg font-medium mb-2">No knowledge bases yet</p>
            <p className="text-muted-foreground mb-4">Create one to start uploading documents.</p>
            <Button onClick={() => setShowCreate(true)}>Create Knowledge Base</Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {knowledgeBases.map((kb) => (
            <Card key={kb.id} className="hover:shadow-md transition-shadow cursor-pointer" onClick={() => setSelectedKB(kb.id)}>
              <CardHeader>
                <CardTitle className="text-lg">{kb.name}</CardTitle>
                <CardDescription>{kb.description || "No description"}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-3 gap-2 text-center">
                  <div>
                    <p className="text-2xl font-bold">{kb.doc_count}</p>
                    <p className="text-xs text-muted-foreground">Documents</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{kb.chunk_count}</p>
                    <p className="text-xs text-muted-foreground">Chunks</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{(kb.total_tokens / 1000).toFixed(0)}K</p>
                    <p className="text-xs text-muted-foreground">Tokens</p>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mt-3">Created {formatRelativeTime(kb.created_at)}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
