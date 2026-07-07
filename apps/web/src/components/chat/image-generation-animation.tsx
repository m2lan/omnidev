"use client";

import { useState, useEffect } from "react";
import { cn } from "@/lib/utils";
import type { Attachment } from "@/lib/api/client";

interface ImageGenerationPlaceholderProps {
  prompt: string;
  progress: "idle" | "generating" | "downloading" | "complete" | "error";
  error?: string;
}

// Animated placeholder shown during image generation
export function ImageGenerationPlaceholder({
  prompt,
  progress,
  error,
}: ImageGenerationPlaceholderProps) {
  const [dots, setDots] = useState("");

  useEffect(() => {
    if (progress === "generating" || progress === "downloading") {
      const interval = setInterval(() => {
        setDots((prev) => (prev.length >= 3 ? "" : prev + "."));
      }, 400);
      return () => clearInterval(interval);
    }
  }, [progress]);

  if (progress === "error") {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
        <div className="flex items-center gap-2 mb-1">
          <span>❌</span>
          <span className="font-medium">Image generation failed</span>
        </div>
        {error && <p className="text-xs opacity-70 mt-1">{error}</p>}
      </div>
    );
  }

  return (
    <div className="image-gen-placeholder rounded-xl overflow-hidden">
      {/* Gradient border animation */}
      <div className="image-gen-border rounded-xl p-[2px]">
        <div className="bg-muted rounded-xl p-4">
          {/* Shimmer block */}
          <div className="relative w-full aspect-square max-w-[400px] rounded-lg overflow-hidden bg-muted-foreground/5 placeholder-breathe">
            {/* Animated gradient overlay */}
            <div className="image-gen-shimmer absolute inset-0" />

            {/* Center icon */}
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-3">
              <div className="image-gen-icon-pulse text-4xl">🎨</div>
              <div className="text-sm text-muted-foreground font-medium">
                {progress === "generating" && `Generating image${dots}`}
                {progress === "downloading" && `Downloading${dots}`}
                {progress === "complete" && "Complete!"}
              </div>
            </div>
          </div>

          {/* Prompt text */}
          <p className="mt-3 text-xs text-muted-foreground line-clamp-2">
            &ldquo;{prompt}&rdquo;
          </p>
        </div>
      </div>
    </div>
  );
}

interface GeneratedImageProps {
  attachment: Attachment;
  prompt?: string;
}

// Generated image with reveal animation
export function GeneratedImage({ attachment, prompt }: GeneratedImageProps) {
  const [loaded, setLoaded] = useState(false);
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <div
        className={cn(
          "generated-image-container cursor-pointer",
          "rounded-xl overflow-hidden border bg-muted/30",
          "transition-all duration-300 hover:shadow-lg",
          expanded ? "max-w-[800px]" : "max-w-[400px]"
        )}
        onClick={() => setExpanded(!expanded)}
      >
        {/* Image with blur-to-clear reveal */}
        <div className="relative overflow-hidden">
          <img
            src={attachment.storage_url}
            alt={prompt || attachment.filename}
            className={cn(
              "w-full h-auto transition-all duration-700 ease-out",
              loaded ? "image-reveal-done" : "image-reveal-loading"
            )}
            onLoad={() => setLoaded(true)}
            loading="lazy"
          />

          {/* Loading overlay */}
          {!loaded && (
            <div className="absolute inset-0 flex items-center justify-center bg-muted/50">
              <div className="text-2xl image-gen-icon-pulse">✨</div>
            </div>
          )}
        </div>

        {/* Prompt caption */}
        {prompt && (
          <div className="px-3 py-2 text-xs text-muted-foreground border-t">
            <p className="line-clamp-2">&ldquo;{prompt}&rdquo;</p>
          </div>
        )}
      </div>

      {/* Expanded overlay */}
      {expanded && (
        <div
          className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-8 cursor-pointer"
          onClick={() => setExpanded(false)}
        >
          <img
            src={attachment.storage_url}
            alt={prompt || attachment.filename}
            className="max-w-full max-h-full object-contain rounded-lg shadow-2xl"
          />
        </div>
      )}
    </>
  );
}
