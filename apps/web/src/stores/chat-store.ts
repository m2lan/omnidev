import { create } from "zustand";
import { api, type Conversation, type Message } from "@/lib/api/client";

interface ChatState {
  conversations: Conversation[];
  activeConversationId: string | null;
  messages: Message[];
  isLoading: boolean;
  isSending: boolean;
  sendingConversationId: string | null;
  streamingContent: string;
  streamingReasoning: string;
  error: string | null;
  selectedModel: string;

  // Actions
  fetchConversations: () => Promise<void>;
  createConversation: (title?: string) => Promise<Conversation>;
  setActiveConversation: (id: string) => Promise<void>;
  deleteConversation: (id: string) => Promise<void>;
  sendMessage: (content: string) => Promise<void>;
  setSelectedModel: (model: string) => void;
  clearError: () => void;
  resetSending: () => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  messages: [],
  isLoading: false,
  isSending: false,
  sendingConversationId: null,
  streamingContent: "",
  streamingReasoning: "",
  error: null,
  selectedModel: "",

  fetchConversations: async () => {
    set({ isLoading: true, error: null });
    try {
      const { data } = await api.listConversations({ page_size: 50 });
      set({ conversations: data || [], isLoading: false });
    } catch (err) {
      set({
        isLoading: false,
        error: err instanceof Error ? err.message : "Failed to load conversations",
      });
    }
  },

  createConversation: async (title?: string) => {
    set({ error: null });
    try {
      const { data } = await api.createConversation({
        title,
        model_id: get().selectedModel || undefined,
      });
      set((state) => ({
        conversations: [data, ...state.conversations],
        activeConversationId: data.id,
        messages: [],
      }));
      return data;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : "Failed to create conversation" });
      throw err;
    }
  },

  setActiveConversation: async (id: string) => {
    const { sendingConversationId } = get();
    set({
      activeConversationId: id,
      messages: [],
      isLoading: true,
      error: null,
      // Clear streaming content if switching to a different conversation
      streamingContent: sendingConversationId === id ? get().streamingContent : "",
      streamingReasoning: sendingConversationId === id ? get().streamingReasoning : "",
    });
    try {
      // Only load last 10 messages for performance
      const { data } = await api.listMessages(id, { page_size: 10 });
      set({ messages: data || [], isLoading: false });
    } catch (err) {
      set({
        isLoading: false,
        error: err instanceof Error ? err.message : "Failed to load messages",
      });
    }
  },

  deleteConversation: async (id: string) => {
    try {
      await api.deleteConversation(id);
      set((state) => {
        const conversations = state.conversations.filter((c) => c.id !== id);
        const activeConversationId =
          state.activeConversationId === id
            ? conversations[0]?.id || null
            : state.activeConversationId;
        return { conversations, activeConversationId };
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : "Failed to delete conversation" });
    }
  },

  sendMessage: async (content: string) => {
    let { activeConversationId } = get();
    const { selectedModel } = get();

    // Create conversation if none exists
    if (!activeConversationId) {
      try {
        const conv = await get().createConversation();
        activeConversationId = conv.id;
      } catch {
        return;
      }
    }

    const conversationId = activeConversationId;
    set({ isSending: true, sendingConversationId: conversationId, error: null, streamingContent: "", streamingReasoning: "" });

    // Optimistic: add user message
    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: conversationId,
      role: "user",
      content,
      created_at: new Date().toISOString(),
    };
    set((state) => ({ messages: [...state.messages, userMsg] }));

    let fullContent = "";
    let fullReasoning = "";

    await api.sendMessageStream(
      conversationId,
      content,
      selectedModel || undefined,
      // onChunk
      (delta: string, type?: string) => {
        if (type === "reasoning") {
          fullReasoning += delta;
        } else {
          fullContent += delta;
        }
        // Only update streaming content if still on same conversation
        if (get().activeConversationId === conversationId) {
          set({ streamingContent: fullContent, streamingReasoning: fullReasoning });
        }
      },
      // onUserMessage
      (msg: Message) => {
        set((state) => ({
          messages: state.messages.map((m) =>
            m.id === userMsg.id ? msg : m
          ),
        }));
      },
      // onComplete
      (assistantMsg: Message) => {
        set((state) => ({
          messages: state.activeConversationId === conversationId
            ? [...state.messages, assistantMsg]
            : state.messages,
          isSending: state.sendingConversationId === conversationId ? false : state.isSending,
          sendingConversationId: state.sendingConversationId === conversationId ? null : state.sendingConversationId,
          streamingContent: state.sendingConversationId === conversationId ? "" : state.streamingContent,
          streamingReasoning: state.sendingConversationId === conversationId ? "" : state.streamingReasoning,
        }));
        // Refresh conversation list
        get().fetchConversations();
      },
      // onError
      (errorMsg: string) => {
        set((state) => ({
          messages: state.activeConversationId === conversationId
            ? state.messages.filter((m) => m.id !== userMsg.id)
            : state.messages,
          isSending: state.sendingConversationId === conversationId ? false : state.isSending,
          sendingConversationId: state.sendingConversationId === conversationId ? null : state.sendingConversationId,
          streamingContent: state.sendingConversationId === conversationId ? "" : state.streamingContent,
          streamingReasoning: state.sendingConversationId === conversationId ? "" : state.streamingReasoning,
          error: errorMsg,
        }));
      }
    );
  },

  setSelectedModel: (model: string) => {
    set({ selectedModel: model });
  },

  clearError: () => {
    set({ error: null });
  },

  resetSending: () => {
    set({ isSending: false, sendingConversationId: null, streamingContent: "", streamingReasoning: "" });
  },
}));
