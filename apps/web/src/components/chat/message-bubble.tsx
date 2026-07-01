"use client";

import { useState, memo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism";
import { cn } from "@/lib/utils";
import type { Message } from "@/lib/api/client";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

interface MessageBubbleProps {
  message: Message;
}

// Pre-process content to wrap HTML documents in code blocks
// Only triggers when content STARTS with a full HTML document
function preprocessContent(content: string): string {
  // Already wrapped in code block
  if (content.trimStart().startsWith('```')) {
    return content;
  }

  const trimmed = content.trimStart();

  // Only detect HTML at the very beginning of content
  const startsWithHtml =
    trimmed.startsWith('<!DOCTYPE') ||
    trimmed.startsWith('<html') ||
    trimmed.startsWith('<HTML');

  if (startsWithHtml) {
    return '```html\n' + content + '\n```';
  }

  return content;
}

export const MessageBubble = memo(function MessageBubble({ message }: MessageBubbleProps) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const isUser = message.role === "user";
  const isStreaming = message.id === "streaming";

  // Truncate long messages for performance
  const MAX_LENGTH = 3000;
  const isLong = !isUser && !isStreaming && message.content.length > MAX_LENGTH;
  const shouldTruncate = isLong && !expanded;

  // Smart truncation: don't cut in middle of code block
  const getTruncatedContent = (content: string, maxLen: number): string => {
    let truncated = content.substring(0, maxLen);
    // Count code block markers
    const codeBlockCount = (truncated.match(/```/g) || []).length;
    // If odd number of code block markers, we're inside a code block
    if (codeBlockCount % 2 !== 0) {
      // Find the last complete code block end
      const lastEnd = truncated.lastIndexOf('```');
      if (lastEnd > 0) {
        truncated = truncated.substring(0, lastEnd + 3);
      }
    }
    return truncated + "\n\n...(truncated)";
  };

  const displayMessage = shouldTruncate
    ? { ...message, content: getTruncatedContent(message.content, MAX_LENGTH) }
    : message;

  const handleCopy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        "flex gap-3 px-4 py-3 animate-fade-in chat-message",
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
      <div className={cn("flex-1 min-w-0 overflow-hidden", isUser ? "flex justify-end" : "")}>
        <div
          className={cn(
            "rounded-lg px-4 py-2.5 max-w-[85%] overflow-hidden break-words overflow-wrap-break-word",
            isUser
              ? "bg-primary text-primary-foreground"
              : "bg-muted"
          )}
        >
          {isUser ? (
            <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          ) : isStreaming ? (
            // Streaming: render as plain text to avoid broken markdown
            <p className="text-sm whitespace-pre-wrap">
              {message.content}
              <span className="inline-block w-2 h-4 bg-foreground animate-pulse ml-0.5" />
            </p>
          ) : (
            <div className="markdown-body text-sm overflow-hidden">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  code({ className, children, node, ...props }) {
                    const match = /language-(\w+)/.exec(className || "");
                    const isInline = !match;
                    const language = match ? match[1] : "";

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
                      <div className="relative group overflow-hidden rounded-lg max-w-full">
                        <div className="absolute right-2 top-2 z-10 flex items-center gap-2">
                          {language && (
                            <span className="text-xs text-muted-foreground bg-background/80 px-2 py-0.5 rounded">
                              {language}
                            </span>
                          )}
                          <button
                            onClick={handleCopy}
                            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-muted-foreground/20 opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            {copied ? "Copied!" : "Copy"}
                          </button>
                        </div>
                        <SyntaxHighlighter
                          language={language || "text"}
                          style={oneDark}
                          customStyle={{
                            margin: 0,
                            borderRadius: "0.5rem",
                            padding: "2.5rem 1rem 1rem",
                            fontSize: "0.85rem",
                            wordBreak: "break-all",
                            whiteSpace: "pre-wrap",
                            maxWidth: "100%",
                            overflow: "hidden",
                          }}
                          wrapLongLines={true}
                          showLineNumbers={false}
                        >
                          {String(children).replace(/\n$/, "")}
                        </SyntaxHighlighter>
                      </div>
                    );
                  },
                }}
              >
                {preprocessContent(displayMessage.content)}
              </ReactMarkdown>
              {isLong && !expanded && (
                <button
                  onClick={() => setExpanded(true)}
                  className="mt-2 text-xs text-primary hover:underline"
                >
                  Show full message ({message.content.length} chars)
                </button>
              )}
            </div>
          )}
        </div>

        {/* Metadata */}
        {!isUser && !isStreaming && (
          <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground">
            {message.model_id && (
              <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5">
                🤖 {message.model_id}
              </span>
            )}
            {message.token_input !== undefined && message.token_input !== null && (
              <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5">
                📥 {message.token_input} in
              </span>
            )}
            {message.token_output !== undefined && message.token_output !== null && (
              <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5">
                📤 {message.token_output} out
              </span>
            )}
            {message.latency_ms !== undefined && message.latency_ms !== null && (
              <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5">
                ⏱️ {message.latency_ms < 1000
                  ? `${message.latency_ms}ms`
                  : `${(message.latency_ms / 1000).toFixed(1)}s`}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
});
