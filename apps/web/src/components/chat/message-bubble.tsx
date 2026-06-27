"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";
import type { Message } from "@/lib/api/client";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

interface MessageBubbleProps {
  message: Message;
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const [copied, setCopied] = useState(false);

  const isUser = message.role === "user";

  const handleCopy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        "flex gap-3 px-4 py-3 animate-fade-in",
        isUser ? "flex-row-reverse" : ""
      )}
    >
      {/* Avatar */}
      <Avatar className="h-8 w-8 shrink-0">
        <AvatarFallback
          className={cn(
            "text-xs font-medium",
            isUser
              ? "bg-primary text-primary-foreground"
              : "bg-muted text-muted-foreground"
          )}
        >
          {isUser ? "U" : "AI"}
        </AvatarFallback>
      </Avatar>

      {/* Content */}
      <div className={cn("flex-1 min-w-0", isUser ? "text-right" : "")}>
        <div
          className={cn(
            "inline-block rounded-lg px-4 py-2.5 max-w-full",
            isUser
              ? "bg-primary text-primary-foreground"
              : "bg-muted"
          )}
        >
          {isUser ? (
            <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          ) : (
            <div className="markdown-body text-sm">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  code({ className, children, ...props }) {
                    const match = /language-(\w+)/.exec(className || "");
                    const isInline = !match;

                    if (isInline) {
                      return (
                        <code
                          className="rounded bg-muted-foreground/20 px-1.5 py-0.5 text-xs"
                          {...props}
                        >
                          {children}
                        </code>
                      );
                    }

                    return (
                      <div className="relative">
                        <div className="absolute right-2 top-2">
                          <button
                            onClick={handleCopy}
                            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-muted-foreground/20"
                          >
                            {copied ? "Copied!" : "Copy"}
                          </button>
                        </div>
                        <pre className="overflow-x-auto rounded-lg bg-background p-4">
                          <code className={className} {...props}>
                            {children}
                          </code>
                        </pre>
                      </div>
                    );
                  },
                }}
              >
                {message.content}
              </ReactMarkdown>
            </div>
          )}
        </div>

        {/* Metadata */}
        <div
          className={cn(
            "flex items-center gap-2 mt-1 text-xs text-muted-foreground",
            isUser ? "justify-end" : ""
          )}
        >
          {message.model_id && <span>{message.model_id}</span>}
          {message.token_output && (
            <span>{message.token_output} tokens</span>
          )}
          {message.latency_ms && <span>{message.latency_ms}ms</span>}
        </div>
      </div>
    </div>
  );
}
