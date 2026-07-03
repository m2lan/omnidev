"use client";

import { useState, useRef, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { api, type Attachment } from "@/lib/api/client";

interface FileUploadButtonProps {
  onUploadComplete: (attachment: Attachment) => void;
  onError: (error: string) => void;
  disabled?: boolean;
}

const ALLOWED_TYPES = [
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "text/plain",
  "text/markdown",
];

const MAX_FILE_SIZE = 20 * 1024 * 1024; // 20MB

export function FileUploadButton({
  onUploadComplete,
  onError,
  disabled,
}: FileUploadButtonProps) {
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      console.log("File selected:", e.target.files);
      const files = e.target.files;
      if (!files || files.length === 0) {
        console.log("No files selected");
        return;
      }

      const fileArray = Array.from(files);
      console.log("File array:", fileArray);
      console.log("First file:", fileArray[0]);
      console.log("File type:", fileArray[0]?.type);
      console.log("Allowed types:", ALLOWED_TYPES);
      console.log("Is allowed:", ALLOWED_TYPES.includes(fileArray[0]?.type));

      // Reset input so same file can be selected again
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }

      // Process each file
      for (const file of fileArray) {
        console.log("Processing file:", file.name, file.type, file.size);

        // Validate file type
        if (!ALLOWED_TYPES.includes(file.type)) {
          console.log("Invalid file type:", file.type);
          onError(
            `File type "${file.type || "unknown"}" is not supported. Allowed: images, PDF, Word, text files.`
          );
          continue;
        }

        // Validate file size
        if (file.size > MAX_FILE_SIZE) {
          console.log("File too large:", file.size);
          onError(
            `File "${file.name}" is too large (${formatFileSize(file.size)}). Maximum size is 20MB.`
          );
          continue;
        }

        console.log("Starting upload...");
        setIsUploading(true);
        try {
          const { data } = await api.uploadFile(file);
          console.log("Upload complete:", data);
          onUploadComplete(data);
        } catch (err) {
          console.error("Upload error:", err);
          onError(
            err instanceof Error ? err.message : `Failed to upload "${file.name}"`
          );
        } finally {
          setIsUploading(false);
        }
      }
    },
    [onUploadComplete, onError]
  );

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        multiple
        accept={ALLOWED_TYPES.join(",")}
        onChange={handleFileSelect}
        className="hidden"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={() => fileInputRef.current?.click()}
        disabled={disabled || isUploading}
        title="Attach file (images, PDF, Word, text)"
        className="shrink-0"
      >
        {isUploading ? (
          <LoadingSpinner />
        ) : (
          <PaperclipIcon />
        )}
      </Button>
    </>
  );
}

// Attachment preview list
interface AttachmentPreviewListProps {
  attachments: Attachment[];
  onRemove: (id: string) => void;
  disabled?: boolean;
}

export function AttachmentPreviewList({
  attachments,
  onRemove,
  disabled,
}: AttachmentPreviewListProps) {
  if (attachments.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 mb-2">
      {attachments.map((att) => (
        <AttachmentPreview
          key={att.id}
          attachment={att}
          onRemove={() => onRemove(att.id)}
          disabled={disabled}
        />
      ))}
    </div>
  );
}

interface AttachmentPreviewProps {
  attachment: Attachment;
  onRemove: () => void;
  disabled?: boolean;
}

function AttachmentPreview({
  attachment,
  onRemove,
  disabled,
}: AttachmentPreviewProps) {
  const isImage = attachment.mime_type.startsWith("image/");

  return (
    <div className="relative group flex items-center gap-2 rounded-lg border bg-muted/50 px-3 py-2 text-sm max-w-[200px]">
      {isImage && (
        <div className="w-8 h-8 rounded overflow-hidden shrink-0">
          <img
            src={attachment.storage_url}
            alt={attachment.filename}
            className="w-full h-full object-cover"
          />
        </div>
      )}
      {!isImage && (
        <div className="w-8 h-8 rounded bg-muted flex items-center justify-center shrink-0">
          <FileIcon mimeType={attachment.mime_type} />
        </div>
      )}
      <div className="flex-1 min-w-0">
        <div className="truncate text-xs font-medium">
          {attachment.filename}
        </div>
        <div className="text-xs text-muted-foreground">
          {formatFileSize(attachment.file_size)}
        </div>
      </div>
      <button
        type="button"
        onClick={onRemove}
        disabled={disabled}
        className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-destructive text-destructive-foreground flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity text-xs"
        title="Remove"
      >
        ×
      </button>
    </div>
  );
}

// Icons
function PaperclipIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48" />
    </svg>
  );
}

function LoadingSpinner() {
  return (
    <svg
      className="animate-spin"
      xmlns="http://www.w3.org/2000/svg"
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    >
      <path d="M21 12a9 9 0 1 1-6.219-8.56" />
    </svg>
  );
}

function FileIcon({ mimeType }: { mimeType: string }) {
  if (mimeType.includes("pdf")) {
    return <span className="text-xs">📄</span>;
  }
  if (mimeType.includes("word") || mimeType.includes("document")) {
    return <span className="text-xs">📝</span>;
  }
  return <span className="text-xs">📎</span>;
}

// Utility
function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}
