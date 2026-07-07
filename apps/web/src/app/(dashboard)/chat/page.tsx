"use client";

import { useEffect } from "react";
import { useChatStore } from "@/stores/chat-store";
import { ConversationList } from "@/components/chat/conversation-list";
import { ChatArea } from "@/components/chat/chat-area";
import { Button } from "@/components/ui/button";
import type { GenerateImageParams } from "@/lib/api/client";

export default function ChatPage() {
  const {
    conversations,
    activeConversationId,
    messages,
    isLoading,
    error,
    selectedModel,
    sendingConversationIds,
    streamingStates,
    imageGeneration,
    _scrollToBottom,
    _loadingComplete,
    fetchConversations,
    createConversation,
    setActiveConversation,
    deleteConversation,
    sendMessage,
    generateImage,
    setSelectedModel,
    clearError,
    resetSending,
  } = useChatStore();

  // Compute per-conversation states
  const isActiveSending = activeConversationId ? sendingConversationIds.has(activeConversationId) : false;
  const activeStreaming = activeConversationId ? streamingStates[activeConversationId] : undefined;
  const streamingContent = activeStreaming?.content || "";
  const streamingReasoning = activeStreaming?.reasoning || "";

  useEffect(() => {
    resetSending();
    fetchConversations();
  }, [fetchConversations, resetSending]);

  const handleNewChat = () => {
    useChatStore.setState({
      activeConversationId: null,
      messages: [],
      error: null,
    });
  };

  const handleSelectConversation = (id: string) => {
    setActiveConversation(id);
  };

  const handleDeleteConversation = async (id: string) => {
    await deleteConversation(id);
  };

  const handleSendMessage = async (content: string, attachmentIds?: string[], attachments?: import("@/lib/api/client").Attachment[]) => {
    await sendMessage(content, attachmentIds, attachments);
  };

  const handleGenerateImage = async (params: GenerateImageParams) => {
    await generateImage(params);
  };

  return (
    <div className="flex h-full min-h-0">
      {/* Conversation List */}
      <div className="w-72 border-r flex flex-col min-h-0">
        <div className="p-3 border-b shrink-0">
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
          sendingIds={sendingConversationIds}
        />
      </div>

      {/* Chat Area */}
      <div className="flex-1 flex flex-col min-h-0">
        <ChatArea
          messages={messages}
          isLoading={isLoading}
          isSending={isActiveSending}
          streamingContent={streamingContent}
          streamingReasoning={streamingReasoning}
          error={error}
          selectedModel={selectedModel}
          onSend={handleSendMessage}
          onModelChange={setSelectedModel}
          onClearError={clearError}
          hasConversation={!!activeConversationId}
          imageGeneration={imageGeneration}
          onGenerateImage={handleGenerateImage}
          scrollToBottomTrigger={_scrollToBottom}
          loadingCompleteTrigger={_loadingComplete}
        />
      </div>
    </div>
  );
}
