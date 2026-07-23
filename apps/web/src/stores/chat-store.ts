import { create } from "zustand";
import { api, type Conversation, type Message, type Attachment, type KnowledgeBase, type GenerateImageParams } from "@/lib/api/client";

interface StreamingState {
  content: string;
  reasoning: string;
  a2uiMessages: object[];
}

interface ImageGenerationState {
  isGenerating: boolean;
  prompt: string;
  progress: "idle" | "generating" | "downloading" | "complete" | "error";
  error?: string;
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

  // Image generation state
  imageGeneration: ImageGenerationState;

  // Selected knowledge bases for RAG
  selectedKBs: KnowledgeBase[];

  // Scroll trigger (timestamp to force scroll to bottom)
  _scrollToBottom?: number;

  // Loading complete trigger (timestamp to force scroll after loading)
  _loadingComplete?: number;

  // Actions
  fetchConversations: () => Promise<void>;
  createConversation: (title?: string) => Promise<Conversation>;
  setActiveConversation: (id: string) => Promise<void>;
  deleteConversation: (id: string) => Promise<void>;
  sendMessage: (content: string, attachmentIds?: string[], attachments?: Attachment[], knowledgeBaseIds?: string[]) => Promise<void>;
  generateImage: (params: GenerateImageParams) => Promise<void>;
  setSelectedModel: (model: string) => void;
  setSelectedKBs: (kbs: KnowledgeBase[]) => void;
  toggleKB: (kb: KnowledgeBase) => void;
  removeKB: (kbId: string) => void;
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
  imageGeneration: {
    isGenerating: false,
    prompt: "",
    progress: "idle",
  },
  selectedKBs: [],

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
      const { selectedKBs, selectedModel } = get();
      const { data } = await api.createConversation({
        title,
        model_id: selectedModel || undefined,
        knowledge_base_ids: selectedKBs.length > 0 ? selectedKBs.map((kb) => kb.id) : undefined,
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
    const { activeConversationId, messages, conversations } = get();

    // Save current messages to cache before switching
    if (activeConversationId) {
      set((state) => ({
        messagesCache: { ...state.messagesCache, [activeConversationId]: messages },
      }));
    }

    // Load conversation's knowledge_base_ids into selectedKBs
    const conv = conversations.find((c) => c.id === id);
    if (conv?.knowledge_base_ids && conv.knowledge_base_ids.length > 0) {
      // We need to load KB details for the IDs
      try {
        const { data: allKBs } = await api.listKnowledgeBases({ page_size: 100 });
        const matchedKBs = (allKBs || []).filter((kb) => conv.knowledge_base_ids.includes(kb.id));
        set({ selectedKBs: matchedKBs });
      } catch {
        // If KB loading fails, just clear selectedKBs
        set({ selectedKBs: [] });
      }
    } else {
      set({ selectedKBs: [] });
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
      set({ messages: cached, isLoading: false, _loadingComplete: Date.now() });
      return;
    }

    try {
      const { data } = await api.listMessages(id, { page_size: 50 });
      set({ messages: data || [], isLoading: false, _loadingComplete: Date.now() });
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
      const wasActive = get().activeConversationId === id;
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

      // If the deleted conversation was active, load the new active conversation's messages
      if (wasActive) {
        const { activeConversationId, messagesCache: cache } = get();
        if (activeConversationId) {
          const cached = cache[activeConversationId];
          if (cached) {
            set({ messages: cached, _loadingComplete: Date.now() });
          } else {
            set({ messages: [], isLoading: true });
            try {
              const { data } = await api.listMessages(activeConversationId, { page_size: 50 });
              set({ messages: data || [], isLoading: false, _loadingComplete: Date.now() });
            } catch {
              set({ isLoading: false });
            }
          }
        } else {
          set({ messages: [] });
        }
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : "Failed to delete conversation" });
    }
  },

  sendMessage: async (content: string, attachmentIds?: string[], attachments?: Attachment[], knowledgeBaseIds?: string[]) => {
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
        [conversationId]: { content: "", reasoning: "", a2uiMessages: [] },
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
    let fullA2UIMessages: object[] = [];

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
            [conversationId]: { content: fullContent, reasoning: fullReasoning, a2uiMessages: fullA2UIMessages },
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
        // Attach A2UI messages to the assistant message if any were received
        if (fullA2UIMessages.length > 0) {
          assistantMsg.content_type = "a2ui";
          assistantMsg.a2ui_messages = fullA2UIMessages;
        }
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
      },
      // knowledgeBaseIds
      knowledgeBaseIds,
      // onA2UIChunk
      (messages: object[]) => {
        fullA2UIMessages = [...fullA2UIMessages, ...messages];
        set((state) => ({
          streamingStates: {
            ...state.streamingStates,
            [conversationId]: {
              content: fullContent,
              reasoning: fullReasoning,
              a2uiMessages: fullA2UIMessages,
            },
          },
        }));
      }
    );
  },

  generateImage: async (params: GenerateImageParams) => {
    let { activeConversationId } = get();

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

    // Set generating state
    set({
      imageGeneration: {
        isGenerating: true,
        prompt: params.prompt,
        progress: "generating",
      },
      error: null,
    });

    // Optimistic: show user message immediately
    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: conversationId,
      role: "user",
      content: `🎨 ${params.prompt}`,
      created_at: new Date().toISOString(),
    };
    set((state) => {
      const newMessages = [...state.messages, userMsg];
      return {
        messages: state.activeConversationId === conversationId ? newMessages : state.messages,
        messagesCache: { ...state.messagesCache, [conversationId]: newMessages },
      };
    });

    try {
      const { data: results } = await api.generateImage({
        ...params,
        conversation_id: conversationId,
      });

      set({
        imageGeneration: {
          isGenerating: false,
          prompt: params.prompt,
          progress: "complete",
        },
      });

      // Reload messages from server to get persisted user + assistant messages
      if (results && results.length > 0) {
        try {
          const { data: msgs } = await api.listMessages(conversationId, { page_size: 50 });
          set((state) => ({
            messages: state.activeConversationId === conversationId ? (msgs || []) : state.messages,
            messagesCache: { ...state.messagesCache, [conversationId]: msgs || [] },
            // Trigger scroll to bottom after message reload
            _scrollToBottom: Date.now(),
          }));
        } catch {
          // ignore reload errors
        }

        // Refresh conversations to get updated title
        get().fetchConversations();
      }
    } catch (err) {
      set({
        imageGeneration: {
          isGenerating: false,
          prompt: params.prompt,
          progress: "error",
          error: err instanceof Error ? err.message : "Image generation failed",
        },
        error: err instanceof Error ? err.message : "Image generation failed",
      });
    }
  },

  setSelectedModel: (model: string) => {
    set({ selectedModel: model });
  },

  setSelectedKBs: (kbs: KnowledgeBase[]) => {
    set({ selectedKBs: kbs });
  },

  toggleKB: (kb: KnowledgeBase) => {
    set((state) => {
      const exists = state.selectedKBs.some((k) => k.id === kb.id);
      return {
        selectedKBs: exists
          ? state.selectedKBs.filter((k) => k.id !== kb.id)
          : [...state.selectedKBs, kb],
      };
    });
  },

  removeKB: (kbId: string) => {
    set((state) => ({
      selectedKBs: state.selectedKBs.filter((k) => k.id !== kbId),
    }));
  },

  clearError: () => {
    set({ error: null });
  },

  resetSending: () => {
    set({ sendingConversationIds: new Set(), streamingStates: {} });
  },
}));
