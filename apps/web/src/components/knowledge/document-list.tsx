"use client";

import { type Document, type DocumentStatus } from "@/lib/api/client";
import { formatFileSize, formatRelativeTime } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface DocumentListProps {
  documents: Document[];
  isLoading?: boolean;
  onDelete: (docId: string) => void;
}

const statusConfig: Record<DocumentStatus, { label: string; color: string }> = {
  uploading: { label: "Uploading", color: "bg-blue-100 text-blue-700" },
  processing: { label: "Processing", color: "bg-yellow-100 text-yellow-700" },
  ready: { label: "Ready", color: "bg-green-100 text-green-700" },
  failed: { label: "Failed", color: "bg-red-100 text-red-700" },
};

const fileTypeIcons: Record<string, string> = {
  pdf: "📄",
  docx: "📝",
  pptx: "📊",
  xlsx: "📈",
  md: "📋",
  txt: "📃",
};

export function DocumentList({ documents, isLoading, onDelete }: DocumentListProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="h-16 rounded-lg bg-muted animate-pulse" />
        ))}
      </div>
    );
  }

  if (documents.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <p className="text-4xl mb-2">📂</p>
        <p>No documents uploaded yet</p>
        <p className="text-sm mt-1">Upload documents to build your knowledge base.</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {/* Header */}
      <div className="grid grid-cols-[1fr_80px_100px_80px_100px_40px] gap-4 px-4 py-2 text-xs font-medium text-muted-foreground uppercase tracking-wider">
        <span>File</span>
        <span>Type</span>
        <span>Size</span>
        <span>Chunks</span>
        <span>Status</span>
        <span></span>
      </div>

      {/* Rows */}
      {documents.map((doc) => {
        const status = statusConfig[doc.status] || statusConfig.ready;
        return (
          <div
            key={doc.id}
            className="grid grid-cols-[1fr_80px_100px_80px_100px_40px] gap-4 items-center px-4 py-3 rounded-lg border hover:bg-muted/50 transition-colors"
            >
            {/* Filename */}
            <div className="flex items-center gap-2 min-w-0">
              <span className="text-lg flex-shrink-0">{fileTypeIcons[doc.file_type] || "📄"}</span>
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{doc.filename}</p>
                <p className="text-xs text-muted-foreground">{formatRelativeTime(doc.created_at)}</p>
              </div>
            </div>

            {/* Type */}
            <span className="text-xs bg-muted px-2 py-0.5 rounded w-fit uppercase">{doc.file_type}</span>

            {/* Size */}
            <span className="text-sm text-muted-foreground">{formatFileSize(doc.file_size)}</span>

            {/* Chunks */}
            <span className="text-sm">{doc.status === "ready" ? doc.chunk_count : "—"}</span>

            {/* Status */}
            <span className={`text-xs px-2 py-0.5 rounded w-fit ${status.color}`}>
              {status.label}
            </span>

            {/* Actions */}
            <Button
              variant="ghost"
              size="sm"
              className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive"
              onClick={() => onDelete(doc.id)}
              title="Delete document"
            >
              ✕
            </Button>
          </div>
        );
      })}

      {/* Error details for failed documents */}
      {documents.filter((d) => d.status === "failed" && d.error).map((doc) => (
        <div key={`err-${doc.id}`} className="mx-4 p-2 rounded bg-red-50 border border-red-200 text-xs text-red-700">
          <strong>{doc.filename}:</strong> {doc.error}
        </div>
      ))}
    </div>
  );
}
