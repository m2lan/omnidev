import { create } from "zustand";
import { api, type Conversation, type Message } from "@/lib/api/client";

interface ChatState {
  conversations: Conversation[];
  activeConversationId: string | null;
  messages: Message[];
  isLoading: boolean;
  isSending: boolean;
  streamingContent: string;
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
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  messages: [],
  isLoading: false,
  isSending: false,
  streamingContent: "",
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
    set({ activeConversationId: id, messages: [], isLoading: true, error: null });
    try {
      const { data } = await api.listMessages(id, { page_size: 100 });
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
    const { activeConversationId, selectedModel } = get();
    if (!activeConversationId) return;

    set({ isSending: true, error: null, streamingContent: "" });

    // Optimistic: add user message
    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: activeConversationId,
      role: "user",
      content,
      created_at: new Date().toISOString(),
    };
    set((state) => ({ messages: [...state.messages, userMsg] }));

    let fullContent = "";

    await api.sendMessageStream(
      activeConversationId,
      content,
      selectedModel || undefined,
      // onChunk
      (delta: string) => {
        fullContent += delta;
        set({ streamingContent: fullContent });
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
          messages: [
            ...state.messages,
            assistantMsg,
          ],
          isSending: false,
          streamingContent: "",
        }));
        // Refresh conversation list
        get().fetchConversations();
      },
      // onError
      (errorMsg: string) => {
        set((state) => ({
          messages: state.messages.filter((m) => m.id !== userMsg.id),
          isSending: false,
          streamingContent: "",
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
}));
