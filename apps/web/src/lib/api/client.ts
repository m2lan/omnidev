// API Client for OmniDev Platform

function getApiBase(): string {
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  // In browser, use same host with gateway port
  if (typeof window !== "undefined") {
    const { protocol, hostname } = window.location;
    return `${protocol}//${hostname}:9090`;
  }
  return "http://localhost:9090";
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

class ApiClient {
  private accessToken: string | null = null;
  private refreshTokenPromise: Promise<string> | null = null;

  private get baseUrl(): string {
    return getApiBase();
  }

  // Lazy getter: always read from localStorage to handle page refresh
  private getToken(): string | null {
    if (this.accessToken) return this.accessToken;
    if (typeof window !== "undefined") {
      this.accessToken = localStorage.getItem("access_token");
    }
    return this.accessToken;
  }

  setAccessToken(token: string | null) {
    this.accessToken = token;
    if (typeof window !== "undefined") {
      if (token) {
        localStorage.setItem("access_token", token);
      } else {
        localStorage.removeItem("access_token");
      }
    }
  }

  getAccessToken(): string | null {
    return this.getToken();
  }

  // Refresh access token using refresh token
  private async refreshAccessToken(): Promise<string> {
    // Deduplicate concurrent refresh calls
    if (this.refreshTokenPromise) {
      return this.refreshTokenPromise;
    }

    this.refreshTokenPromise = (async () => {
      const refreshToken = localStorage.getItem("refresh_token");
      if (!refreshToken) {
        throw new Error("No refresh token available");
      }

      const response = await fetch(`${this.baseUrl}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!response.ok) {
        // Refresh failed - clear everything
        this.setAccessToken(null);
        localStorage.removeItem("refresh_token");
        throw new Error("Token refresh failed");
      }

      const { data } = await response.json();
      this.setAccessToken(data.access_token);
      localStorage.setItem("refresh_token", data.refresh_token);
      return data.access_token;
    })();

    try {
      return await this.refreshTokenPromise;
    } finally {
      this.refreshTokenPromise = null;
    }
  }

  // Check if JWT token is expired (with 30s buffer)
  private isTokenExpired(token: string): boolean {
    try {
      const payload = JSON.parse(atob(token.split(".")[1]));
      return payload.exp * 1000 < Date.now() + 30000;
    } catch {
      return true;
    }
  }

  private async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { body, headers: customHeaders, ...rest } = options;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...(customHeaders as Record<string, string>),
    };

    let token = this.getToken();

    // Proactively refresh if token is about to expire
    if (token && this.isTokenExpired(token)) {
      try {
        token = await this.refreshAccessToken();
      } catch {
        // Refresh failed, will get 401 below and redirect
      }
    }

    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const doFetch = async (): Promise<Response> => {
      return fetch(`${this.baseUrl}${path}`, {
        ...rest,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
    };

    let response = await doFetch();

    // If 401 and we have a token, try refreshing
    if (response.status === 401 && token) {
      try {
        const newToken = await this.refreshAccessToken();
        headers["Authorization"] = `Bearer ${newToken}`;
        response = await doFetch();
      } catch {
        // Refresh failed, redirect to login
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
      }
    }

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        error: { code: response.status, message: response.statusText },
      }));
      throw new ApiError(
        error.error?.message || "Request failed",
        error.error?.code || response.status,
        error.error?.detail
      );
    }

    return response.json();
  }

  // Auth
  async register(data: RegisterInput) {
    return this.post<ApiResponse<AuthResponse>>("/api/v1/auth/register", data);
  }

  async login(data: LoginInput) {
    return this.post<ApiResponse<AuthResponse>>("/api/v1/auth/login", data);
  }

  async refreshToken(refreshToken: string) {
    return this.post<ApiResponse<TokenResponse>>("/api/v1/auth/refresh", {
      refresh_token: refreshToken,
    });
  }

  // User
  async getProfile() {
    return this.get<ApiResponse<User>>("/api/v1/users/me");
  }

  async updateProfile(data: UpdateProfileInput) {
    return this.patch<ApiResponse<User>>("/api/v1/users/me", data);
  }

  // API Keys
  async listAPIKeys() {
    return this.get<ApiResponse<APIKey[]>>("/api/v1/users/me/api-keys");
  }

  async createAPIKey(data: CreateAPIKeyInput) {
    return this.post<ApiResponse<{ api_key: APIKey; key: string }>>("/api/v1/users/me/api-keys", data);
  }

  async revokeAPIKey(id: string) {
    return this.delete<ApiResponse<void>>(`/api/v1/users/me/api-keys/${id}`);
  }

  // Conversations
  async listConversations(params?: ListParams) {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", String(params.page));
    if (params?.page_size) query.set("page_size", String(params.page_size));
    if (params?.search) query.set("search", params.search);
    return this.get<ApiResponse<Conversation[]>>(`/api/v1/conversations?${query}`);
  }

  async createConversation(data: CreateConversationInput) {
    return this.post<ApiResponse<Conversation>>("/api/v1/conversations", data);
  }

  async getConversation(id: string) {
    return this.get<ApiResponse<Conversation>>(`/api/v1/conversations/${id}`);
  }

  async deleteConversation(id: string) {
    return this.delete<ApiResponse<void>>(`/api/v1/conversations/${id}`);
  }

  async listMessages(conversationId: string, params?: ListParams) {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", String(params.page));
    if (params?.page_size) query.set("page_size", String(params.page_size));
    return this.get<ApiResponse<Message[]>>(
      `/api/v1/conversations/${conversationId}/messages?${query}`
    );
  }

  async sendMessage(conversationId: string, content: string, modelId?: string) {
    return this.post<ApiResponse<{ user_message: Message; assistant_message: Message }>>(
      `/api/v1/conversations/${conversationId}/messages`,
      { content, model_id: modelId }
    );
  }

  // Streaming send
  async sendMessageStream(
    conversationId: string,
    content: string,
    modelId: string | undefined,
    attachmentIds: string[] | undefined,
    onChunk: (delta: string, type?: string) => void,
    onUserMessage: (msg: Message) => void,
    onComplete: (assistantMsg: Message) => void,
    onError: (error: string) => void,
    knowledgeBaseIds?: string[]
  ) {
    const token = this.getToken();
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    try {
      const requestBody: Record<string, unknown> = {
        content,
        model_id: modelId,
        attachment_ids: attachmentIds,
      };
      if (knowledgeBaseIds && knowledgeBaseIds.length > 0) {
        requestBody.knowledge_base_ids = knowledgeBaseIds;
      }
      console.log("Sending stream request:", requestBody);
      const response = await fetch(
        `${this.baseUrl}/api/v1/conversations/${conversationId}/messages/stream`,
        {
          method: "POST",
          headers,
          body: JSON.stringify(requestBody),
        }
      );

      if (!response.ok) {
        const error = await response.json().catch(() => ({
          error: { message: response.statusText },
        }));
        onError(error.error?.message || "Request failed");
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        onError("No response body");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        let currentEvent = "";
        for (const line of lines) {
          if (line.startsWith("event: ")) {
            currentEvent = line.slice(7).trim();
            continue;
          }
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            if (data === "{}") continue;

            try {
              const parsed = JSON.parse(data);

              // Complete message from gateway
              if (currentEvent === "complete") {
                onComplete(parsed);
                currentEvent = "";
                continue;
              }

              // Check for user_message event
              if (parsed.id && parsed.role === "user") {
                onUserMessage(parsed);
                currentEvent = "";
                continue;
              }

              // Check for delta content
              if (parsed.delta) {
                onChunk(parsed.delta, parsed.type);
                currentEvent = "";
                continue;
              }
            } catch {
              // Ignore parse errors
            }
            currentEvent = "";
          }
        }
      }
    } catch (err) {
      onError(err instanceof Error ? err.message : "Stream failed");
    }
  }

  // Knowledge
  async listKnowledgeBases(params?: ListParams) {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", String(params.page));
    if (params?.page_size) query.set("page_size", String(params.page_size));
    return this.get<ApiResponse<KnowledgeBase[]>>(`/api/v1/knowledge?${query}`);
  }

  async createKnowledgeBase(data: CreateKBInput) {
    return this.post<ApiResponse<KnowledgeBase>>("/api/v1/knowledge", data);
  }

  async searchKnowledgeBase(kbId: string, query: string, topK?: number) {
    return this.post<ApiResponse<SearchResult[]>>(
      `/api/v1/knowledge/${kbId}/search`,
      { query, top_k: topK || 5 }
    );
  }

  async getKnowledgeBase(id: string) {
    return this.get<ApiResponse<KnowledgeBase>>(`/api/v1/knowledge/${id}`);
  }

  async updateKnowledgeBase(id: string, data: Partial<CreateKBInput>) {
    return this.patch<ApiResponse<KnowledgeBase>>(`/api/v1/knowledge/${id}`, data);
  }

  async deleteKnowledgeBase(id: string) {
    return this.delete<ApiResponse<{ message: string }>>(`/api/v1/knowledge/${id}`);
  }

  async listDocuments(kbId: string, params?: ListParams) {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", String(params.page));
    if (params?.page_size) query.set("page_size", String(params.page_size));
    return this.get<ApiResponse<Document[]>>(`/api/v1/knowledge/${kbId}/documents?${query}`);
  }

  async uploadDocument(kbId: string, file: File): Promise<ApiResponse<Document>> {
    const token = this.getToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const formData = new FormData();
    formData.append("file", file);

    const resp = await fetch(`${this.baseUrl}/api/v1/knowledge/${kbId}/documents`, {
      method: "POST",
      headers,
      body: formData,
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      throw new Error(err?.error?.detail || `Upload failed (${resp.status})`);
    }

    return resp.json();
  }

  async deleteDocument(kbId: string, docId: string) {
    return this.delete<ApiResponse<{ message: string }>>(`/api/v1/knowledge/${kbId}/documents/${docId}`);
  }

  // Models
  async listModels() {
    return this.get<ApiResponse<Model[]>>("/api/v1/models");
  }

  // Upload
  async uploadFile(file: File): Promise<ApiResponse<Attachment>> {
    const token = this.getToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const formData = new FormData();
    formData.append("file", file);

    const response = await fetch(`${this.baseUrl}/api/v1/upload`, {
      method: "POST",
      headers,
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        error: { message: response.statusText },
      }));
      throw new ApiError(
        error.error?.message || "Upload failed",
        error.error?.code || response.status
      );
    }

    return response.json();
  }

  async deleteAttachment(id: string) {
    return this.delete<ApiResponse<void>>(`/api/v1/attachments/${id}`);
  }

  async getAttachmentUrl(id: string) {
    return this.get<ApiResponse<{ url: string }>>(`/api/v1/attachments/${id}/url`);
  }

  // Image Generation
  async generateImage(params: GenerateImageParams) {
    return this.post<ApiResponse<GeneratedImageResult[]>>("/api/v1/images/generate", params);
  }

  // User AI Configs
  async listAIConfigs() {
    return this.get<ApiResponse<UserAIConfig[]>>("/api/v1/user/ai-configs");
  }

  async getAIConfig(id: string) {
    return this.get<ApiResponse<UserAIConfig>>(`/api/v1/user/ai-configs/${id}`);
  }

  async createAIConfig(data: CreateAIConfigInput) {
    return this.post<ApiResponse<UserAIConfig>>("/api/v1/user/ai-configs", data);
  }

  async updateAIConfig(id: string, data: UpdateAIConfigInput) {
    return this.put<ApiResponse<UserAIConfig>>(`/api/v1/user/ai-configs/${id}`, data);
  }

  async deleteAIConfig(id: string) {
    return this.delete<ApiResponse<{ message: string }>>(`/api/v1/user/ai-configs/${id}`);
  }

  async setDefaultAIConfig(id: string) {
    return this.put<ApiResponse<{ message: string }>>(`/api/v1/user/ai-configs/${id}/default`, {});
  }

  async testAIConfigConnection(id: string) {
    return this.post<ApiResponse<TestConnectionResult>>(`/api/v1/user/ai-configs/${id}/test`, {});
  }

  // Helpers
  private put<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>(path, { method: "PUT", body });
  }
  private get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "GET" });
  }

  private post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>(path, { method: "POST", body });
  }

  private patch<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>(path, { method: "PATCH", body });
  }

  private delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "DELETE" });
  }
}

export class ApiError extends Error {
  code: number;
  detail?: string;

  constructor(message: string, code: number, detail?: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.detail = detail;
  }
}

// Types
export interface ApiResponse<T> {
  data: T;
  meta?: {
    total_count: number;
    page: number;
    page_size: number;
    next_page_token?: string;
  };
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export interface User {
  id: string;
  email: string;
  nickname: string;
  avatar_url?: string;
  role: string;
  created_at: string;
}

export interface Conversation {
  id: string;
  user_id: string;
  title?: string;
  model_id?: string;
  system_prompt?: string;
  status: string;
  pinned: boolean;
  tags: string[];
  message_count: number;
  created_at: string;
  updated_at: string;
}

export interface Attachment {
  id: string;
  user_id: string;
  conversation_id?: string;
  message_id?: string;
  filename: string;
  mime_type: string;
  file_size: number;
  storage_url: string;
  thumbnail_key?: string;
  width?: number;
  height?: number;
  created_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  model_id?: string;
  token_input?: number;
  token_output?: number;
  latency_ms?: number;
  attachments?: Attachment[];
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface Model {
  id: string;
  provider: string;
  model_id: string;
  display_name: string;
  description?: string;
  context_window: number;
  supports_streaming: boolean;
  supports_vision: boolean;
  supports_tools: boolean;
}

export interface KnowledgeBase {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  embedding_model: string;
  chunk_size: number;
  chunk_overlap: number;
  doc_count: number;
  chunk_count: number;
  total_tokens: number;
  total_size: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export type DocumentStatus = "uploading" | "processing" | "ready" | "failed";

export interface Document {
  id: string;
  knowledge_base_id: string;
  filename: string;
  file_type: string;
  file_size: number;
  file_url: string;
  status: DocumentStatus;
  error?: string;
  chunk_count: number;
  total_tokens: number;
  metadata: Record<string, unknown>;
  processed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchResult {
  chunk: {
    id: string;
    content: string;
    metadata: Record<string, unknown>;
  };
  score: number;
  source: string;
}

export interface ListParams {
  page?: number;
  page_size?: number;
  search?: string;
}

export interface RegisterInput {
  email: string;
  password: string;
  nickname: string;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface UpdateProfileInput {
  nickname?: string;
  avatar_url?: string;
  bio?: string;
}

export interface CreateConversationInput {
  title?: string;
  model_id?: string;
  system_prompt?: string;
  tags?: string[];
}

export interface CreateKBInput {
  name: string;
  description?: string;
  chunk_size?: number;
  chunk_overlap?: number;
}

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  created_at: string;
  expires_at?: string;
}

export interface CreateAPIKeyInput {
  name: string;
  scopes?: string[];
  expires_at?: string;
}

export interface ModelConfig {
  id: string;
  display_name: string;
  default_temperature?: number;
  default_max_tokens?: number;
  context_window?: number;
}

export interface UserAIConfig {
  id: string;
  user_id: string;
  provider: string;
  display_name: string;
  api_key_mask: string;
  base_url: string;
  protocol: "openai" | "anthropic";
  models: ModelConfig[];
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAIConfigInput {
  provider: string;
  display_name: string;
  api_key: string;
  base_url: string;
  protocol: "openai" | "anthropic";
  models: ModelConfig[];
  is_default?: boolean;
}

export interface UpdateAIConfigInput {
  display_name?: string;
  api_key?: string;
  base_url?: string;
  protocol?: "openai" | "anthropic";
  models?: ModelConfig[];
  is_default?: boolean;
  is_active?: boolean;
}

export interface TestConnectionResult {
  success: boolean;
  message: string;
  latency_ms: number;
}

export interface GenerateImageParams {
  conversation_id?: string;
  model: string;
  prompt: string;
  size?: string;
  quality?: string;
  style?: string;
  n?: number;
}

export interface GeneratedImageResult {
  attachment: Attachment;
  revised_prompt?: string;
}

export const api = new ApiClient();
