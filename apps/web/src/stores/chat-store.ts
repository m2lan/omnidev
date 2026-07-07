import { create } from "zustand";
import { api, type Conversation, type Message, type Attachment } from "@/lib/api/client";

interface StreamingState {
  content: string;
  reasoning: string;
}

interface ChatState {
  conversations: Conversation[];
  activeConversationId: string | null;
  messages: Message[];
  messagesCache: Record<string, Message[]>;
  isLoading: boolean;
  error: string | null;
  selectedModel: string;

  // Per-conversation streaming state
  streamingStates: Record<string, StreamingState>;
  sendingConversationIds: Set<string>;

  // Actions
  fetchConversations: () => Promise<void>;
  createConversation: (title?: string) => Promise<Conversation>;
  setActiveConversation: (id: string) => Promise<void>;
  deleteConversation: (id: string) => Promise<void>;
  sendMessage: (content: string, attachmentIds?: string[], attachments?: Attachment[]) => Promise<void>;
  setSelectedModel: (model: string) => void;
  clearError: () => void;
  resetSending: () => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  messages: [],
  messagesCache: {},
  isLoading: false,
  error: null,
  selectedModel: "",
  streamingStates: {},
  sendingConversationIds: new Set(),

  // Computed: is the active conversation sending?
  get isActiveSending() {
    const state = get();
    return state.activeConversationId ? state.sendingConversationIds.has(state.activeConversationId) : false;
  },

  // Computed: streaming content for active conversation
  get activeStreamingContent() {
    const state = get();
    return state.activeConversationId ? (state.streamingStates[state.activeConversationId]?.content || "") : "";
  },

  get activeStreamingReasoning() {
    const state = get();
    return state.activeConversationId ? (state.streamingStates[state.activeConversationId]?.reasoning || "") : "";
  },

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
    const { activeConversationId, messages } = get();

    // Save current messages to cache before switching
    if (activeConversationId) {
      set((state) => ({
        messagesCache: { ...state.messagesCache, [activeConversationId]: messages },
      }));
    }

    set({
      activeConversationId: id,
      messages: [],
      isLoading: true,
      error: null,
    });

    // Try cache first, then server
    const cached = get().messagesCache[id];
    if (cached) {
      set({ messages: cached, isLoading: false });
      return;
    }

    try {
      const { data } = await api.listMessages(id, { page_size: 50 });
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
        const { [id]: _, ...messagesCache } = state.messagesCache;
        const { [id]: __, ...streamingStates } = state.streamingStates;
        const sendingConversationIds = new Set(state.sendingConversationIds);
        sendingConversationIds.delete(id);
        return { conversations, activeConversationId, messagesCache, streamingStates, sendingConversationIds };
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : "Failed to delete conversation" });
    }
  },

  sendMessage: async (content: string, attachmentIds?: string[], attachments?: Attachment[]) => {
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

    // Mark this conversation as sending
    set((state) => ({
      sendingConversationIds: new Set([...state.sendingConversationIds, conversationId]),
      streamingStates: {
        ...state.streamingStates,
        [conversationId]: { content: "", reasoning: "" },
      },
      error: null,
    }));

    // Optimistic: add user message
    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: conversationId,
      role: "user",
      content,
      attachments: attachments && attachments.length > 0 ? attachments : undefined,
      created_at: new Date().toISOString(),
    };
    set((state) => {
      const newMessages = [...state.messages, userMsg];
      return {
        messages: state.activeConversationId === conversationId ? newMessages : state.messages,
        messagesCache: { ...state.messagesCache, [conversationId]: newMessages },
      };
    });

    let fullContent = "";
    let fullReasoning = "";

    await api.sendMessageStream(
      conversationId,
      content,
      selectedModel || undefined,
      attachmentIds,
      // onChunk
      (delta: string, type?: string) => {
        if (type === "reasoning") {
          fullReasoning += delta;
        } else {
          fullContent += delta;
        }
        // Update this conversation's streaming state
        set((state) => ({
          streamingStates: {
            ...state.streamingStates,
            [conversationId]: { content: fullContent, reasoning: fullReasoning },
          },
        }));
      },
      // onUserMessage
      (msg: Message) => {
        set((state) => {
          const cached = state.messagesCache[conversationId] || [];
          const newMessages = cached.map((m) => {
            if (m.id === userMsg.id) {
              // Preserve attachments from the optimistic message
              return { ...msg, attachments: m.attachments || msg.attachments };
            }
            return m;
          });
          return {
            messages: state.activeConversationId === conversationId ? newMessages : state.messages,
            messagesCache: { ...state.messagesCache, [conversationId]: newMessages },
          };
        });
      },
      // onComplete
      (assistantMsg: Message) => {
        set((state) => {
          const cached = state.messagesCache[conversationId] || [];
          const finalMessages = [...cached, assistantMsg];

          // Remove from sending set and clear streaming state
          const sendingConversationIds = new Set(state.sendingConversationIds);
          sendingConversationIds.delete(conversationId);
          const { [conversationId]: _, ...streamingStates } = state.streamingStates;

          // Auto-update conversation title from first user message (matches backend generateTitle)
          const conversations = state.conversations.map((c) => {
            if (c.id === conversationId && (!c.title || c.title === "")) {
              const autoTitle = content.length > 50 ? content.slice(0, 50) + "..." : content;
              return { ...c, title: autoTitle };
            }
            return c;
          });

          return {
            messages: state.activeConversationId === conversationId ? finalMessages : state.messages,
            messagesCache: { ...state.messagesCache, [conversationId]: finalMessages },
            sendingConversationIds,
            streamingStates,
            conversations,
          };
        });
      },
      // onError
      (errorMsg: string) => {
        set((state) => {
          // Remove from sending set and clear streaming state
          const sendingConversationIds = new Set(state.sendingConversationIds);
          sendingConversationIds.delete(conversationId);
          const { [conversationId]: _, ...streamingStates } = state.streamingStates;

          return {
            sendingConversationIds,
            streamingStates,
            error: errorMsg,
          };
        });
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
    set({ sendingConversationIds: new Set(), streamingStates: {} });
  },
}));
