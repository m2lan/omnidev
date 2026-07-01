"use client";

import { useState, useRef, useEffect } from "react";
import type { Message } from "@/lib/api/client";
import { MessageBubble } from "@/components/chat/message-bubble";
import { ModelSelector } from "@/components/chat/model-selector";
import { Button } from "@/components/ui/button";

interface ChatAreaProps {
  messages: Message[];
  isSending: boolean;
  streamingContent: string;
  error: string | null;
  selectedModel: string;
  onSend: (content: string) => Promise<void>;
  onModelChange: (model: string) => void;
  onClearError: () => void;
  hasConversation: boolean;
}

export function ChatArea({
  messages,
  isSending,
  streamingContent,
  error,
  selectedModel,
  onSend,
  onModelChange,
  onClearError,
  hasConversation,
}: ChatAreaProps) {
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streamingContent]);

  // Auto-resize textarea
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = Math.min(textareaRef.current.scrollHeight, 200) + "px";
    }
  }, [input]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isSending) return;

    const content = input.trim();
    setInput("");
    onClearError();
    await onSend(content);
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
                { icon: "🔍", text: "Explain this concept..." },
                { icon: "🐛", text: "Debug this code..." },
              ].map((suggestion) => (
                <button
                  key={suggestion.text}
                  onClick={() => setInput(suggestion.text)}
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
            <ModelSelector value={selectedModel} onChange={onModelChange} />
            <form onSubmit={handleSubmit} className="flex gap-2 mt-2">
              <textarea
                ref={textareaRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Type your message... (Shift+Enter for new line)"
                className="flex-1 resize-none rounded-lg border bg-background px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                rows={1}
                disabled={isSending}
              />
              <Button type="submit" disabled={!input.trim() || isSending} className="self-end">
                {isSending ? "..." : "Send"}
              </Button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Messages */}
      <div className="flex-1 overflow-y-auto">
        <div className="w-full mx-auto py-6 px-4">
          {messages.map((msg) => (
            <MessageBubble key={msg.id} message={msg} />
          ))}

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
          {isSending && !streamingContent && (
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
          <ModelSelector value={selectedModel} onChange={onModelChange} />
          <form onSubmit={handleSubmit} className="flex gap-2 mt-2">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type your message... (Shift+Enter for new line)"
              className="flex-1 resize-none rounded-lg border bg-background px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              rows={1}
              disabled={isSending}
            />
            <Button type="submit" disabled={!input.trim() || isSending} className="self-end">
              {isSending ? "..." : "Send"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
