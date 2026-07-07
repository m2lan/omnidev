"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import type { Message, Attachment, GenerateImageParams } from "@/lib/api/client";
import { MessageBubble } from "@/components/chat/message-bubble";
import { ModelSelector } from "@/components/chat/model-selector";
import {
  FileUploadButton,
  AttachmentPreviewList,
} from "@/components/chat/file-upload-button";
import { ImageGenerationPlaceholder } from "@/components/chat/image-generation-animation";
import { Button } from "@/components/ui/button";

interface ChatAreaProps {
  messages: Message[];
  isLoading: boolean;
  isSending: boolean;
  streamingContent: string;
  streamingReasoning: string;
  error: string | null;
  selectedModel: string;
  onSend: (content: string, attachmentIds?: string[], attachments?: Attachment[]) => Promise<void>;
  onModelChange: (model: string) => void;
  onClearError: () => void;
  hasConversation: boolean;
  // Image generation props
  imageGeneration?: {
    isGenerating: boolean;
    prompt: string;
    progress: "idle" | "generating" | "downloading" | "complete" | "error";
    error?: string;
  };
  onGenerateImage?: (params: GenerateImageParams) => Promise<void>;
  // Scroll trigger
  scrollToBottomTrigger?: number;
  // Loading complete trigger (set after messages loaded and isLoading becomes false)
  loadingCompleteTrigger?: number;
}

export function ChatArea({
  messages,
  isLoading,
  isSending,
  streamingContent,
  streamingReasoning,
  error,
  selectedModel,
  onSend,
  onModelChange,
  onClearError,
  hasConversation,
  imageGeneration,
  onGenerateImage,
  scrollToBottomTrigger,
  loadingCompleteTrigger,
}: ChatAreaProps) {
  const [input, setInput] = useState("");
  const [pendingAttachments, setPendingAttachments] = useState<Attachment[]>([]);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [isImageMode, setIsImageMode] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const justSentRef = useRef(false);
  const prevMsgCountRef = useRef(0);

  const scrollToBottom = useCallback((force = false) => {
    const container = scrollContainerRef.current;
    if (!container) return;
    if (!force) {
      const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
      if (!isNearBottom) return;
    }
    container.scrollTop = container.scrollHeight;
  }, []);

  // Force scroll when messages first load (0 → N) or user just sent
  useEffect(() => {
    const prevCount = prevMsgCountRef.current;
    prevMsgCountRef.current = messages.length;

    // First load or user just sent: force scroll
    if (prevCount === 0 && messages.length > 0) {
      return;
    }
    if (justSentRef.current) {
      justSentRef.current = false;
      scrollToBottom(true);
      return;
    }
    // Otherwise smart scroll
    scrollToBottom(false);
  }, [messages, streamingContent, imageGeneration?.isGenerating, scrollToBottom]);

  // Force scroll when loading completes or image generation finishes
  useEffect(() => {
    if (!loadingCompleteTrigger && !scrollToBottomTrigger) return;
    if (messages.length === 0) return;

    // Multi-pass scroll: rAF for layout, then delayed retries for lazy content
    let cancelled = false;
    const scrollNow = () => {
      const container = scrollContainerRef.current;
      if (container && container.scrollHeight > container.clientHeight) {
        container.scrollTop = container.scrollHeight;
      }
    };
    // Double rAF: wait for layout + paint
    requestAnimationFrame(() => requestAnimationFrame(() => {
      if (!cancelled) scrollNow();
    }));
    // Fallback: markdown/code rendering may finish later
    const t1 = setTimeout(() => { if (!cancelled) scrollNow(); }, 150);
    const t2 = setTimeout(() => { if (!cancelled) scrollNow(); }, 400);
    return () => { cancelled = true; clearTimeout(t1); clearTimeout(t2); };
  }, [loadingCompleteTrigger, scrollToBottomTrigger, messages.length]);

  // Auto-resize textarea
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = Math.min(textareaRef.current.scrollHeight, 200) + "px";
    }
  }, [input]);

  const handleUploadComplete = useCallback((attachment: Attachment) => {
    console.log("Upload complete callback:", attachment);
    setPendingAttachments((prev) => {
      console.log("Updating pending attachments:", prev, attachment);
      return [...prev, attachment];
    });
    setUploadError(null);
  }, []);

  const handleUploadError = useCallback((error: string) => {
    setUploadError(error);
    // Auto-clear after 5 seconds
    setTimeout(() => setUploadError(null), 5000);
  }, []);

  const handleRemoveAttachment = useCallback((id: string) => {
    setPendingAttachments((prev) => prev.filter((a) => a.id !== id));
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const hasContent = input.trim().length > 0;
    const hasAttachments = pendingAttachments.length > 0;
    console.log("Submit:", { hasContent, hasAttachments, pendingAttachments });
    if ((!hasContent && !hasAttachments) || isSending) return;

    const content = input.trim();
    justSentRef.current = true;

    // Image generation mode
    if (isImageMode && onGenerateImage && hasContent) {
      setInput("");
      setUploadError(null);
      onClearError();

      try {
        await onGenerateImage({
          model: selectedModel,
          prompt: content,
          size: "1024x1024",
        });
      } catch {
        setInput(content);
      }
      return;
    }

    // Normal chat mode
    const attachmentIds = pendingAttachments.map((a) => a.id);
    const attachmentsData = [...pendingAttachments];
    console.log("Sending with attachmentIds:", attachmentIds, attachmentsData);

    // Clear input and attachments immediately
    setInput("");
    setPendingAttachments([]);
    setUploadError(null);
    onClearError();

    try {
      await onSend(content || "(file attachment)", attachmentIds, attachmentsData);
    } catch {
      // Restore input on error so user can retry
      setInput(content);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  // Empty state
  if (!hasConversation || (messages.length === 0 && !isSending)) {
    return (
      <div className="flex-1 flex flex-col">
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center max-w-md">
            <div className="text-6xl mb-6">✨</div>
            <h2 className="text-2xl font-semibold mb-3">How can I help you today?</h2>
            <p className="text-muted-foreground mb-8">
              Ask me anything about coding, writing, analysis, or any other task.
            </p>

            <div className="grid grid-cols-2 gap-3">
              {[
                { icon: "💻", text: "Write a Python script to..." },
                { icon: "📝", text: "Help me write an email..." },
                { icon: "🎨", text: "Generate an image of...", mode: "image" as const },
                { icon: "🐛", text: "Debug this code..." },
              ].map((suggestion) => (
                <button
                  key={suggestion.text}
                  onClick={() => {
                    setInput(suggestion.text);
                    if (suggestion.mode === "image") {
                      setIsImageMode(true);
                    }
                  }}
                  className="flex items-center gap-2 rounded-lg border p-3 text-left text-sm hover:bg-muted transition-colors"
                >
                  <span>{suggestion.icon}</span>
                  <span className="text-muted-foreground">{suggestion.text}</span>
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Input */}
        <div className="border-t p-4">
          <div className="w-full mx-auto">
            <div className="flex items-center gap-2 mb-2">
              <ModelSelector value={selectedModel} onChange={onModelChange} />
              <button
                type="button"
                onClick={() => setIsImageMode(!isImageMode)}
                className={`ml-auto flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs transition-colors ${
                  isImageMode
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                }`}
                title={isImageMode ? "Switch to Chat mode" : "Switch to Image Generation mode"}
              >
                <span>{isImageMode ? "💬" : "🎨"}</span>
                <span>{isImageMode ? "Chat" : "Image"}</span>
              </button>
            </div>
            {/* Upload error */}
            {uploadError && (
              <div className="mt-2 text-xs text-destructive bg-destructive/10 px-3 py-1.5 rounded">
                {uploadError}
              </div>
            )}
            {/* Attachment previews */}
            {!isImageMode && (
              <AttachmentPreviewList
                attachments={pendingAttachments}
                onRemove={handleRemoveAttachment}
                disabled={isSending}
              />
            )}
            <form onSubmit={handleSubmit} className="flex gap-2 mt-2">
              {!isImageMode && (
                <FileUploadButton
                  onUploadComplete={handleUploadComplete}
                  onError={handleUploadError}
                  disabled={isSending}
                />
              )}
              <textarea
                ref={textareaRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={
                  isImageMode
                    ? "Describe the image you want to generate..."
                    : "Type your message... (Shift+Enter for new line)"
                }
                className="flex-1 resize-none rounded-lg border bg-background px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                rows={1}
                disabled={isSending}
              />
              <Button
                type="submit"
                disabled={(!input.trim() && pendingAttachments.length === 0) || isSending}
                className="self-end"
              >
                {isSending
                  ? "..."
                  : isImageMode
                  ? "🎨 Generate"
                  : "Send"}
              </Button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
        <div className="flex-1 overflow-y-auto overflow-x-hidden p-4">
          <div className="w-full max-w-full space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="flex gap-3 animate-pulse">
                <div className="h-8 w-8 rounded-full bg-muted" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-muted rounded w-3/4" />
                  <div className="h-4 bg-muted rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
      {/* Messages */}
      <div ref={scrollContainerRef} className="flex-1 overflow-y-auto overflow-x-hidden">
        <div className="w-full mx-auto py-6 px-4 max-w-full break-words overflow-hidden">
          {/* Load more hint */}
          {messages.length >= 10 && (
            <div className="text-center mb-4">
              <span className="text-xs text-muted-foreground bg-muted px-3 py-1 rounded-full">
                Showing last 10 messages
              </span>
            </div>
          )}
          {messages.map((msg) => (
            <MessageBubble key={msg.id} message={msg} />
          ))}

          {/* Streaming reasoning (thinking process) */}
          {isSending && streamingReasoning && (
            <div className="flex gap-3 px-4 py-3 animate-fade-in">
              <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center text-muted-foreground text-xs font-medium">
                💭
              </div>
              <div className="flex-1">
                <details open className="text-sm text-muted-foreground">
                  <summary className="cursor-pointer font-medium mb-1">Thinking...</summary>
                  <div className="whitespace-pre-wrap text-xs opacity-70">{streamingReasoning}</div>
                </details>
              </div>
            </div>
          )}

          {/* Streaming content */}
          {isSending && streamingContent && (
            <MessageBubble
              message={{
                id: "streaming",
                conversation_id: "",
                role: "assistant",
                content: streamingContent,
                created_at: new Date().toISOString(),
              }}
            />
          )}

          {/* Loading indicator */}
          {isSending && !streamingContent && !streamingReasoning && !imageGeneration?.isGenerating && (
            <div className="flex gap-3 px-4 py-3 animate-fade-in">
              <div className="h-8 w-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-xs font-medium">
                AI
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-1 text-muted-foreground">
                  <span className="animate-bounce">●</span>
                  <span className="animate-bounce" style={{ animationDelay: "0.1s" }}>●</span>
                  <span className="animate-bounce" style={{ animationDelay: "0.2s" }}>●</span>
                </div>
              </div>
            </div>
          )}

          {/* Image generation placeholder */}
          {imageGeneration?.isGenerating && (
            <div className="flex gap-3 px-4 py-3 animate-fade-in">
              <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center text-muted-foreground text-xs font-medium">
                AI
              </div>
              <div className="flex-1">
                <ImageGenerationPlaceholder
                  prompt={imageGeneration.prompt}
                  progress={imageGeneration.progress}
                  error={imageGeneration.error}
                />
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="border-t border-destructive/50 bg-destructive/10 px-4 py-3">
          <div className="w-full mx-auto flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-destructive">
              <span>⚠️</span>
              <span>{error}</span>
            </div>
            <button
              onClick={onClearError}
              className="text-destructive hover:text-destructive/80 text-sm"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Input */}
      <div className="border-t p-4">
        <div className="w-full mx-auto">
          <div className="flex items-center gap-2 mb-2">
            <ModelSelector value={selectedModel} onChange={onModelChange} />
            <button
              type="button"
              onClick={() => setIsImageMode(!isImageMode)}
              className={`ml-auto flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs transition-colors ${
                isImageMode
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
              title={isImageMode ? "Switch to Chat mode" : "Switch to Image Generation mode"}
            >
              <span>{isImageMode ? "💬" : "🎨"}</span>
              <span>{isImageMode ? "Chat" : "Image"}</span>
            </button>
          </div>
          {/* Upload error */}
          {uploadError && (
            <div className="mt-2 text-xs text-destructive bg-destructive/10 px-3 py-1.5 rounded">
              {uploadError}
            </div>
          )}
          {/* Attachment previews */}
          {!isImageMode && (
            <AttachmentPreviewList
              attachments={pendingAttachments}
              onRemove={handleRemoveAttachment}
              disabled={isSending}
            />
          )}
          <form onSubmit={handleSubmit} className="flex gap-2 mt-2">
            {!isImageMode && (
              <FileUploadButton
                onUploadComplete={handleUploadComplete}
                onError={handleUploadError}
                disabled={isSending}
              />
            )}
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={
                isImageMode
                  ? "Describe the image you want to generate..."
                  : "Type your message... (Shift+Enter for new line)"
              }
              className="flex-1 resize-none rounded-lg border bg-background px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              rows={1}
              disabled={isSending || imageGeneration?.isGenerating}
            />
            <Button
              type="submit"
              disabled={
                !input.trim() ||
                isSending ||
                imageGeneration?.isGenerating
              }
              className="self-end"
            >
              {isSending || imageGeneration?.isGenerating
                ? "..."
                : isImageMode
                ? "🎨 Generate"
                : "Send"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
