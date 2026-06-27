"use client";

import { useEffect } from "react";
import { useChatStore } from "@/stores/chat-store";
import { ConversationList } from "@/components/chat/conversation-list";
import { ChatArea } from "@/components/chat/chat-area";
import { Button } from "@/components/ui/button";

export default function ChatPage() {
  const {
    conversations,
    activeConversationId,
    messages,
    isLoading,
    isSending,
    fetchConversations,
    createConversation,
    setActiveConversation,
    deleteConversation,
    sendMessage,
  } = useChatStore();

  useEffect(() => {
    fetchConversations();
  }, [fetchConversations]);

  const handleNewChat = async () => {
    await createConversation();
  };

  const handleSelectConversation = (id: string) => {
    setActiveConversation(id);
  };

  const handleDeleteConversation = async (id: string) => {
    await deleteConversation(id);
  };

  const handleSendMessage = async (content: string) => {
    if (!activeConversationId) {
      const conv = await createConversation();
      await sendMessage(content);
    } else {
      await sendMessage(content);
    }
  };

  return (
    <div className="flex h-full">
      {/* Conversation List */}
      <div className="w-72 border-r flex flex-col">
        <div className="p-3 border-b">
          <Button onClick={handleNewChat} className="w-full" variant="outline">
            + New Chat
          </Button>
        </div>
        <ConversationList
          conversations={conversations}
          activeId={activeConversationId}
          onSelect={handleSelectConversation}
          onDelete={handleDeleteConversation}
          isLoading={isLoading}
        />
      </div>

      {/* Chat Area */}
      <div className="flex-1 flex flex-col">
        <ChatArea
          messages={messages}
          isSending={isSending}
          onSend={handleSendMessage}
          hasConversation={!!activeConversationId}
        />
      </div>
    </div>
  );
}
