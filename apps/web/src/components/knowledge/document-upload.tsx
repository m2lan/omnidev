"use client";

import { useCallback, useRef, useState } from "react";
import { Button } from "@/components/ui/button";

const ACCEPTED_TYPES = [
  ".pdf", ".docx", ".pptx", ".xlsx", ".md", ".txt",
  "application/pdf",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "text/markdown",
  "text/plain",
];

const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB

interface DocumentUploadProps {
  onUpload: (file: File) => Promise<void>;
  isUploading?: boolean;
}

export function DocumentUpload({ onUpload, isUploading }: DocumentUploadProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateFile = (file: File): string | null => {
    if (file.size > MAX_FILE_SIZE) {
      return `File too large (${(file.size / 1024 / 1024).toFixed(1)}MB). Max 50MB.`;
    }
    const ext = "." + file.name.split(".").pop()?.toLowerCase();
    if (!ACCEPTED_TYPES.includes(ext) && !ACCEPTED_TYPES.includes(file.type)) {
      return `Unsupported file type: ${ext}`;
    }
    return null;
  };

  const handleFiles = useCallback(async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    const file = files[0];
    const err = validateFile(file);
    if (err) {
      setError(err);
      return;
    }
    setError(null);
    try {
      await onUpload(file);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Upload failed");
    }
  }, [onUpload]);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    handleFiles(e.dataTransfer.files);
  }, [handleFiles]);

  return (
    <div className="space-y-2">
      <div
        className={`
          relative border-2 border-dashed rounded-lg p-6 text-center transition-colors cursor-pointer
          ${dragOver ? "border-primary bg-primary/5" : "border-muted-foreground/25 hover:border-muted-foreground/50"}
          ${isUploading ? "opacity-50 pointer-events-none" : ""}
        `}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
      >
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          accept={ACCEPTED_TYPES.join(",")}
          onChange={(e) => handleFiles(e.target.files)}
        />
        <div className="space-y-1">
          <p className="text-4xl">📤</p>
          <p className="text-sm font-medium">
            {isUploading ? "Uploading..." : "Drop file here or click to browse"}
          </p>
          <p className="text-xs text-muted-foreground">
            PDF, DOCX, PPTX, XLSX, MD, TXT — max 50MB
          </p>
        </div>
      </div>

      {error && (
        <p className="text-xs text-red-600">{error}</p>
      )}
    </div>
  );
}
