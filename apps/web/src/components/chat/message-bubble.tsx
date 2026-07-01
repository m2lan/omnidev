"use client";

import { useState, useCallback, useRef, memo } from "react";
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

// Pre-process markdown content for proper code block rendering
function preprocessContent(content: string): string {
  // Fix inline code fences: ``` not at line start → insert newline before it
  // AI sometimes outputs "...text```html\n..." without a line break
  content = content.replace(/([^\n])```(\w*)\n/g, '$1\n```$2\n');

  // Already starts with a code block
  if (content.trimStart().startsWith('```')) {
    return content;
  }

  const trimmed = content.trimStart();

  // Full HTML document at the start → wrap in code block
  if (
    trimmed.startsWith('<!DOCTYPE') ||
    trimmed.startsWith('<html') ||
    trimmed.startsWith('<HTML')
  ) {
    return '```html\n' + content + '\n```';
  }

  return content;
}

// Check if HTML content is a complete document or a fragment
function isCompleteHtmlDocument(html: string): boolean {
  const trimmed = html.trimStart().toLowerCase();
  return trimmed.startsWith('<!doctype') || trimmed.startsWith('<html');
}

// Wrap HTML fragment in a complete document structure
function wrapHtmlFragment(html: string): string {
  if (isCompleteHtmlDocument(html)) {
    return html;
  }
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 16px; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
  </style>
</head>
<body>
${html}
</body>
</html>`;
}

// Sandboxed iframe preview component for HTML content
function HtmlPreview({ html }: { html: string }) {
  const srcdoc = wrapHtmlFragment(html);

  return (
    <div className="w-full bg-white rounded-b-lg overflow-hidden" style={{ minHeight: 200 }}>
      <iframe
        srcDoc={srcdoc}
        sandbox="allow-scripts allow-same-origin"
        className="w-full border-0"
        style={{ minHeight: 200, height: 'auto' }}
        title="HTML Preview"
        onLoad={(e) => {
          // Auto-resize iframe to content height
          const iframe = e.currentTarget;
          try {
            const doc = iframe.contentDocument || iframe.contentWindow?.document;
            if (doc) {
              const height = Math.max(
                doc.documentElement.scrollHeight,
                doc.body.scrollHeight,
                200
              );
              iframe.style.height = `${Math.min(height, 600)}px`;
            }
          } catch {
            // Cross-origin restrictions - use fixed height
            iframe.style.height = '400px';
          }
        }}
      />
    </div>
  );
}

export const MessageBubble = memo(function MessageBubble({ message }: MessageBubbleProps) {
  const [expanded, setExpanded] = useState(false);
  // Track active tab per HTML code block (index -> 'code' | 'preview')
  const [htmlTabs, setHtmlTabs] = useState<Record<number, 'code' | 'preview'>>({});
  // Track copied state per code block (index -> boolean)
  const [copiedBlocks, setCopiedBlocks] = useState<Record<number, boolean>>({});
  // Ref for counting code blocks during render (avoids re-render loop)
  const codeBlockCounterRef = useRef(0);

  const toggleHtmlTab = useCallback((index: number, tab: 'code' | 'preview') => {
    setHtmlTabs(prev => ({ ...prev, [index]: tab }));
  }, []);

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

  // Reset code block counter at start of each render
  codeBlockCounterRef.current = 0;

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

                    const isHtml = language === "html";
                    // Use ref counter for unique index per code block
                    const blockIndex = codeBlockCounterRef.current;
                    codeBlockCounterRef.current++;
                    const activeTab = htmlTabs[blockIndex] || "code";
                    const htmlContent = String(children).replace(/\n$/, "");
                    const isBlockCopied = copiedBlocks[blockIndex] || false;

                    const handleBlockCopy = () => {
                      navigator.clipboard.writeText(htmlContent);
                      setCopiedBlocks(prev => ({ ...prev, [blockIndex]: true }));
                      setTimeout(() => {
                        setCopiedBlocks(prev => ({ ...prev, [blockIndex]: false }));
                      }, 2000);
                    };

                    return (
                      <div className="relative group overflow-hidden rounded-lg max-w-full">
                        <div className="absolute right-2 top-2 z-10 flex items-center gap-2">
                          {isHtml && (
                            <div className="flex items-center bg-background/80 rounded overflow-hidden">
                              <button
                                onClick={() => toggleHtmlTab(blockIndex, 'code')}
                                className={cn(
                                  "px-2 py-0.5 text-xs transition-colors",
                                  activeTab === 'code'
                                    ? "bg-primary text-primary-foreground"
                                    : "text-muted-foreground hover:bg-muted-foreground/20"
                                )}
                              >
                                Code
                              </button>
                              <button
                                onClick={() => toggleHtmlTab(blockIndex, 'preview')}
                                className={cn(
                                  "px-2 py-0.5 text-xs transition-colors",
                                  activeTab === 'preview'
                                    ? "bg-primary text-primary-foreground"
                                    : "text-muted-foreground hover:bg-muted-foreground/20"
                                )}
                              >
                                Preview
                              </button>
                            </div>
                          )}
                          {language && (
                            <span className="text-xs text-muted-foreground bg-background/80 px-2 py-0.5 rounded">
                              {language}
                            </span>
                          )}
                          <button
                            onClick={handleBlockCopy}
                            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-muted-foreground/20 opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            {isBlockCopied ? "Copied!" : "Copy"}
                          </button>
                        </div>
                        {isHtml && activeTab === 'preview' ? (
                          <HtmlPreview html={htmlContent} />
                        ) : isHtml ? (
                          // Lightweight pre for HTML code view (SyntaxHighlighter is too slow for large HTML)
                          <pre
                            style={{
                              margin: 0,
                              borderRadius: "0.5rem",
                              padding: "2.5rem 1rem 1rem",
                              fontSize: "0.85rem",
                              wordBreak: "break-all",
                              whiteSpace: "pre-wrap",
                              maxWidth: "100%",
                              overflow: "hidden",
                              backgroundColor: "#282c34",
                              color: "#abb2bf",
                              fontFamily: "var(--mono, monospace)",
                            }}
                          >{htmlContent}</pre>
                        ) : (
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
                            {htmlContent}
                          </SyntaxHighlighter>
                        )}
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
