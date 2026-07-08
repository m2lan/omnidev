"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, type KnowledgeBase, type Document } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { DocumentList } from "@/components/knowledge/document-list";
import { DocumentUpload } from "@/components/knowledge/document-upload";
import { formatRelativeTime } from "@/lib/utils";

export default function KnowledgeDetailPage() {
  const params = useParams();
  const router = useRouter();
  const kbId = params.id as string;

  const [kb, setKB] = useState<KnowledgeBase | null>(null);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isUploading, setIsUploading] = useState(false);
  const [showUpload, setShowUpload] = useState(false);
  const [totalDocs, setTotalDocs] = useState(0);

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<unknown[]>([]);
  const [isSearching, setIsSearching] = useState(false);

  const loadKB = useCallback(async () => {
    try {
      const { data } = await api.getKnowledgeBase(kbId);
      setKB(data);
    } catch (err) {
      console.error("Failed to load knowledge base:", err);
    }
  }, [kbId]);

  const loadDocuments = useCallback(async () => {
    try {
      const { data, meta } = await api.listDocuments(kbId, { page_size: 100 });
      setDocuments(data || []);
      setTotalDocs(meta?.total_count || 0);
    } catch (err) {
      console.error("Failed to load documents:", err);
    }
  }, [kbId]);

  useEffect(() => {
    Promise.all([loadKB(), loadDocuments()]).finally(() => setIsLoading(false));
  }, [loadKB, loadDocuments]);

  // Poll while documents are processing
  useEffect(() => {
    const hasProcessing = documents.some((d) => d.status === "uploading" || d.status === "processing");
    if (!hasProcessing) return;

    const timer = setInterval(async () => {
      await Promise.all([loadDocuments(), loadKB()]);
    }, 3000);

    return () => clearInterval(timer);
  }, [documents, loadDocuments, loadKB]);

  const handleUpload = async (file: File) => {
    setIsUploading(true);
    try {
      await api.uploadDocument(kbId, file);
      // Reload documents and KB stats
      await Promise.all([loadDocuments(), loadKB()]);
      setShowUpload(false);
    } finally {
      setIsUploading(false);
    }
  };

  const handleDelete = async (docId: string) => {
    if (!confirm("Delete this document? This cannot be undone.")) return;
    try {
      await api.deleteDocument(kbId, docId);
      await Promise.all([loadDocuments(), loadKB()]);
    } catch (err) {
      console.error("Failed to delete document:", err);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setIsSearching(true);
    try {
      const { data } = await api.searchKnowledgeBase(kbId, searchQuery);
      setSearchResults(data || []);
    } catch (err) {
      console.error("Search failed:", err);
    } finally {
      setIsSearching(false);
    }
  };

  if (isLoading) {
    return (
      <div className="p-6 max-w-6xl mx-auto space-y-6">
        <div className="h-8 w-48 bg-muted animate-pulse rounded" />
        <div className="h-32 bg-muted animate-pulse rounded-lg" />
        <div className="h-64 bg-muted animate-pulse rounded-lg" />
      </div>
    );
  }

  if (!kb) {
    return (
      <div className="p-6 max-w-6xl mx-auto text-center py-12">
        <p className="text-4xl mb-4">❌</p>
        <p className="text-lg font-medium mb-2">Knowledge base not found</p>
        <Button variant="outline" onClick={() => router.push("/knowledge")}>
          Back to Knowledge Bases
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <Button variant="ghost" size="sm" className="mb-1 -ml-2" onClick={() => router.push("/knowledge")}>
            ← Back
          </Button>
          <h1 className="text-2xl font-bold">{kb.name}</h1>
          {kb.description && <p className="text-muted-foreground">{kb.description}</p>}
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        {[
          { label: "Documents", value: kb.doc_count },
          { label: "Chunks", value: kb.chunk_count },
          { label: "Tokens", value: `${(kb.total_tokens / 1000).toFixed(0)}K` },
          { label: "Chunk Size", value: kb.chunk_size },
          { label: "Updated", value: formatRelativeTime(kb.updated_at) },
        ].map((stat) => (
          <Card key={stat.label}>
            <CardContent className="py-3 text-center">
              <p className="text-xl font-bold">{stat.value}</p>
              <p className="text-xs text-muted-foreground">{stat.label}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Documents Section */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-lg">Documents ({totalDocs})</CardTitle>
          <Button size="sm" onClick={() => setShowUpload(!showUpload)}>
            {showUpload ? "Cancel" : "+ Upload"}
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {showUpload && (
            <DocumentUpload onUpload={handleUpload} isUploading={isUploading} />
          )}
          <DocumentList documents={documents} onDelete={handleDelete} />
        </CardContent>
      </Card>

      {/* Search Section */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Search This Knowledge Base</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search query..."
              onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              className="flex-1"
            />
            <Button onClick={handleSearch} disabled={isSearching}>
              {isSearching ? "Searching..." : "Search"}
            </Button>
          </div>

          {searchResults.length > 0 && (
            <div className="space-y-2">
              {(searchResults as { score: number; source: string; chunk: { content: string; heading?: string } }[]).map((result, i) => (
                <div key={i} className="rounded-lg border p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-muted-foreground">
                      Score: {result.score?.toFixed(3)}
                    </span>
                    <span className="text-xs bg-muted px-2 py-0.5 rounded">{result.source}</span>
                  </div>
                  {result.chunk?.heading && (
                    <p className="text-xs font-medium text-muted-foreground mb-1">
                      {result.chunk.heading}
                    </p>
                  )}
                  <p className="text-sm">{result.chunk?.content?.substring(0, 300)}...</p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
