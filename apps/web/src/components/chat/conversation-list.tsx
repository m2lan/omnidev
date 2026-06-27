"use client";

import { cn, formatRelativeTime, truncate } from "@/lib/utils";
import type { Conversation } from "@/lib/api/client";

interface ConversationListProps {
  conversations: Conversation[];
  activeId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  isLoading: boolean;
}

export function ConversationList({
  conversations,
  activeId,
  onSelect,
  onDelete,
  isLoading,
}: ConversationListProps) {
  if (isLoading) {
    return (
      <div className="flex-1 p-4">
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-16 rounded-lg bg-muted animate-pulse" />
          ))}
        </div>
      </div>
    );
  }

  if (conversations.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-4 text-center">
        <div>
          <p className="text-2xl mb-2">💬</p>
          <p className="text-sm text-muted-foreground">No conversations yet</p>
          <p className="text-xs text-muted-foreground mt-1">Start a new chat to begin</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto p-2">
      <div className="space-y-1">
        {conversations.map((conv) => (
          <button
            key={conv.id}
            onClick={() => onSelect(conv.id)}
            className={cn(
              "w-full text-left rounded-lg px-3 py-2.5 transition-colors group",
              activeId === conv.id
                ? "bg-accent text-accent-foreground"
                : "hover:bg-muted"
            )}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {conv.title || "New Conversation"}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {conv.message_count} messages • {formatRelativeTime(conv.updated_at)}
                </p>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(conv.id);
                }}
                className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive ml-2"
                title="Delete"
              >
                ✕
              </button>
            </div>
            {conv.tags.length > 0 && (
              <div className="flex gap-1 mt-1.5">
                {conv.tags.slice(0, 3).map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </button>
        ))}
      </div>
    </div>
  );
}
