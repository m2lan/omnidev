import { create } from "zustand";
import { api, type Conversation, type Message } from "@/lib/api/client";

interface ChatState {
  conversations: Conversation[];
  activeConversationId: string | null;
  messages: Message[];
  isLoading: boolean;
  isSending: boolean;

  // Actions
  fetchConversations: () => Promise<void>;
  createConversation: (title?: string) => Promise<Conversation>;
  setActiveConversation: (id: string) => Promise<void>;
  deleteConversation: (id: string) => Promise<void>;
  sendMessage: (content: string, modelId?: string) => Promise<void>;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  messages: [],
  isLoading: false,
  isSending: false,

  fetchConversations: async () => {
    set({ isLoading: true });
    try {
      const { data } = await api.listConversations({ page_size: 50 });
      set({ conversations: data || [], isLoading: false });
    } catch {
      set({ isLoading: false });
    }
  },

  createConversation: async (title?: string) => {
    const { data } = await api.createConversation({ title });
    set((state) => ({
      conversations: [data, ...state.conversations],
      activeConversationId: data.id,
      messages: [],
    }));
    return data;
  },

  setActiveConversation: async (id: string) => {
    set({ activeConversationId: id, messages: [], isLoading: true });
    try {
      const { data } = await api.listMessages(id, { page_size: 100 });
      set({ messages: data || [], isLoading: false });
    } catch {
      set({ isLoading: false });
    }
  },

  deleteConversation: async (id: string) => {
    await api.deleteConversation(id);
    set((state) => {
      const conversations = state.conversations.filter((c) => c.id !== id);
      const activeConversationId =
        state.activeConversationId === id
          ? conversations[0]?.id || null
          : state.activeConversationId;
      return { conversations, activeConversationId };
    });
  },

  sendMessage: async (content: string, modelId?: string) => {
    const { activeConversationId } = get();
    if (!activeConversationId) return;

    set({ isSending: true });

    // Optimistic: add user message
    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: activeConversationId,
      role: "user",
      content,
      created_at: new Date().toISOString(),
    };
    set((state) => ({ messages: [...state.messages, userMsg] }));

    try {
      const { data } = await api.sendMessage(activeConversationId, content, modelId);

      // Replace temp user message and add assistant message
      set((state) => ({
        messages: [
          ...state.messages.filter((m) => m.id !== userMsg.id),
          data.user_message,
          data.assistant_message,
        ],
        isSending: false,
      }));

      // Refresh conversation list to update title/message_count
      get().fetchConversations();
    } catch {
      // Remove optimistic message on error
      set((state) => ({
        messages: state.messages.filter((m) => m.id !== userMsg.id),
        isSending: false,
      }));
    }
  },
}));
