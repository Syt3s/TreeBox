import type {
  AddTenantMemberRequest,
  AddTenantMemberResponse,
  AnswerQuestionRequest,
  AnswerQuestionResponse,
  CreateQuestionRequest,
  CreateQuestionResponse,
  CreateWorkspaceRequest,
  CreateWorkspaceResponse,
  DeactivateResponse,
  DeleteQuestionResponse,
  ExportDataResponse,
  GetQuestionResponse,
  GetQuestionsResponse,
  GetUserResponse,
  GetWorkspaceQuestionStatsResponse,
  ListTenantMembersResponse,
  ListTenantsResponse,
  ListWorkspaceQuestionsResponse,
  ListWorkspacesResponse,
  LoginRequest,
  LoginResponse,
  MarkAllQuestionsViewedResponse,
  MarkQuestionViewedResponse,
  QuestionStatsResponse,
  RegisterRequest,
  RegisterResponse,
  RemoveTenantMemberResponse,
  SetQuestionPrivateResponse,
  SetWorkspaceIntakeResponse,
  UploadUserAssetResponse,
  UpdateTenantMemberRoleRequest,
  UpdateTenantMemberRoleResponse,
  UpdateHarassmentRequest,
  UpdateHarassmentResponse,
  UpdateWorkspaceQuestionAssigneeRequest,
  UpdateWorkspaceQuestionInternalNoteRequest,
  UpdateWorkspaceQuestionPrivacyRequest,
  UpdateProfileRequest,
  UpdateProfileResponse,
  UpdateWorkspaceQuestionStatusRequest,
  User,
  WorkspaceQuestionMutationResponse,
} from "@/types"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL?.trim() || ""

export class ApiError extends Error {
  status: number
  code?: number

  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

function getStoredToken() {
  if (typeof window === "undefined") {
    return ""
  }
  return localStorage.getItem("token")?.trim() || ""
}

function storeToken(token?: string) {
  if (typeof window === "undefined") {
    return
  }

  if (token?.trim()) {
    localStorage.setItem("token", token.trim())
    return
  }

  localStorage.removeItem("token")
}

function clearStoredToken() {
  storeToken()
}

function buildApiUrl(endpoint: string) {
  if (!API_BASE_URL) {
    return endpoint
  }
  return `${API_BASE_URL}${endpoint}`
}

function encodePathSegment(value: string | number) {
  return encodeURIComponent(String(value).trim())
}

async function parseResponseError(response: Response) {
  const fallbackMessage = response.status === 401 ? "登录状态已失效，请重新登录" : "请求失败，请稍后重试"
  const text = await response.text().catch(() => "")

  if (!text) {
    return new ApiError(fallbackMessage, response.status)
  }

  try {
    const error = JSON.parse(text) as { code?: number; message?: string }
    return new ApiError(error.message || fallbackMessage, response.status, error.code)
  } catch {
    return new ApiError(text || fallbackMessage, response.status)
  }
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const method = (options.method || "GET").toUpperCase()
  const hasBody = options.body != null
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string> | undefined),
  }

  if (hasBody && !isFormData && method !== "GET" && method !== "HEAD") {
    headers["Content-Type"] = headers["Content-Type"] || "application/json"
  }

  const token = getStoredToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(buildApiUrl(endpoint), {
    ...options,
    headers,
    credentials: "include",
  })

  if (!response.ok) {
    if (response.status === 401 || response.status === 403) {
      clearStoredToken()
    }
    throw await parseResponseError(response)
  }

  const result = (await response.json()) as
    | { code?: number; message?: string; data?: T }
    | T

  if (result && typeof result === "object" && "code" in result) {
    if (result.code !== 0) {
      throw new ApiError(result.message || "请求失败，请稍后重试", response.status, result.code)
    }
    return result.data as T
  }

  return result as T
}

export const api = {
  auth: {
    login: async (data: LoginRequest) => {
      const res = await request<LoginResponse>("/api/v2/auth/login", {
        method: "POST",
        body: JSON.stringify(data),
      })
      storeToken(res.token)
      return res
    },

    register: async (data: RegisterRequest) => {
      const res = await request<RegisterResponse>("/api/v2/auth/register", {
        method: "POST",
        body: JSON.stringify(data),
      })
      storeToken(res.token)
      return res
    },

    logout: async () => {
      const res = await request<{ success: boolean }>("/api/v2/auth/logout", {
        method: "POST",
      })
      clearStoredToken()
      return res
    },

    getCurrentUser: () => request<User>("/api/v2/auth/me"),
  },

  users: {
    get: (domain: string) => request<GetUserResponse>(`/api/v2/users/${encodePathSegment(domain)}`),
  },

  tenants: {
    list: () => request<ListTenantsResponse>("/api/v2/tenants"),

    members: {
      list: (tenantUid: string) =>
        request<ListTenantMembersResponse>(`/api/v2/tenants/${encodePathSegment(tenantUid)}/members`),

      add: (tenantUid: string, data: AddTenantMemberRequest) =>
        request<AddTenantMemberResponse>(`/api/v2/tenants/${encodePathSegment(tenantUid)}/members`, {
          method: "POST",
          body: JSON.stringify(data),
        }),

      updateRole: (tenantUid: string, memberUserId: number, data: UpdateTenantMemberRoleRequest) =>
        request<UpdateTenantMemberRoleResponse>(
          `/api/v2/tenants/${encodePathSegment(tenantUid)}/members/${encodePathSegment(memberUserId)}/role`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),

      remove: (tenantUid: string, memberUserId: number) =>
        request<RemoveTenantMemberResponse>(
          `/api/v2/tenants/${encodePathSegment(tenantUid)}/members/${encodePathSegment(memberUserId)}`,
          {
            method: "DELETE",
          }
        ),
    },
  },

  workspaces: {
    list: () => request<ListWorkspacesResponse>("/api/v2/workspaces"),

    create: (data: CreateWorkspaceRequest) =>
      request<CreateWorkspaceResponse>("/api/v2/workspaces", {
        method: "POST",
        body: JSON.stringify(data),
      }),

    setIntake: (workspaceUid: string) =>
      request<SetWorkspaceIntakeResponse>(`/api/v2/workspaces/${encodePathSegment(workspaceUid)}/intake`, {
        method: "POST",
      }),

    stats: (workspaceUid: string) =>
      request<GetWorkspaceQuestionStatsResponse>(`/api/v2/workspaces/${encodePathSegment(workspaceUid)}/stats`),

    questions: {
      list: (
        workspaceUid: string,
        params?: {
          page_size?: number
          cursor?: string
          filter_answered?: boolean
          show_private?: boolean
          status?: string
          assigned_to_user_id?: number
          only_assigned?: boolean
          only_unassigned?: boolean
        }
      ) => {
        const searchParams = new URLSearchParams()
        if (params?.page_size) {
          searchParams.set("page_size", params.page_size.toString())
        }
        if (params?.cursor) {
          searchParams.set("cursor", params.cursor)
        }
        if (params?.filter_answered) {
          searchParams.set("filter_answered", "true")
        }
        if (params?.show_private === false) {
          searchParams.set("show_private", "false")
        }
        if (params?.status?.trim()) {
          searchParams.set("status", params.status.trim())
        }
        if (typeof params?.assigned_to_user_id === "number") {
          searchParams.set("assigned_to_user_id", params.assigned_to_user_id.toString())
        }
        if (params?.only_assigned) {
          searchParams.set("only_assigned", "true")
        }
        if (params?.only_unassigned) {
          searchParams.set("only_unassigned", "true")
        }

        const query = searchParams.toString()
        return request<ListWorkspaceQuestionsResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions${query ? `?${query}` : ""}`
        )
      },

      answer: (workspaceUid: string, questionId: number, data: AnswerQuestionRequest) =>
        request<WorkspaceQuestionMutationResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions/${encodePathSegment(questionId)}/answer`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),

      updateStatus: (workspaceUid: string, questionId: number, data: UpdateWorkspaceQuestionStatusRequest) =>
        request<WorkspaceQuestionMutationResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions/${encodePathSegment(questionId)}/status`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),

      updateAssignee: (workspaceUid: string, questionId: number, data: UpdateWorkspaceQuestionAssigneeRequest) =>
        request<WorkspaceQuestionMutationResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions/${encodePathSegment(questionId)}/assignee`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),

      updateInternalNote: (
        workspaceUid: string,
        questionId: number,
        data: UpdateWorkspaceQuestionInternalNoteRequest
      ) =>
        request<WorkspaceQuestionMutationResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions/${encodePathSegment(questionId)}/internal-note`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),

      updatePrivacy: (
        workspaceUid: string,
        questionId: number,
        data: UpdateWorkspaceQuestionPrivacyRequest
      ) =>
        request<WorkspaceQuestionMutationResponse>(
          `/api/v2/workspaces/${encodePathSegment(workspaceUid)}/questions/${encodePathSegment(questionId)}/privacy`,
          {
            method: "POST",
            body: JSON.stringify(data),
          }
        ),
    },
  },

  questions: {
    create: (domain: string, data: CreateQuestionRequest) =>
      request<CreateQuestionResponse>(`/api/v2/questions/${encodePathSegment(domain)}`, {
        method: "POST",
        body: JSON.stringify(data),
      }),

    list: (domain: string, params?: { page_size?: number; cursor?: string }) => {
      const searchParams = new URLSearchParams()
      if (params?.page_size) {
        searchParams.set("page_size", params.page_size.toString())
      }
      if (params?.cursor) {
        searchParams.set("cursor", params.cursor)
      }

      const query = searchParams.toString()
      return request<GetQuestionsResponse>(
        `/api/v2/questions/${encodePathSegment(domain)}${query ? `?${query}` : ""}`
      )
    },

    get: (domain: string, questionId: number) => request<GetQuestionResponse>(
      `/api/v2/questions/${encodePathSegment(domain)}/${encodePathSegment(questionId)}`
    ),

    answer: (domain: string, questionId: number, data: AnswerQuestionRequest) =>
      request<AnswerQuestionResponse>(
        `/api/v2/questions/${encodePathSegment(domain)}/${encodePathSegment(questionId)}/answer`,
        {
        method: "POST",
        body: JSON.stringify(data),
        }
      ),

    delete: (domain: string, questionId: number) =>
      request<DeleteQuestionResponse>(
        `/api/v2/questions/${encodePathSegment(domain)}/${encodePathSegment(questionId)}/delete`,
        {
        method: "POST",
        }
      ),

    setPrivate: (domain: string, questionId: number) =>
      request<SetQuestionPrivateResponse>(
        `/api/v2/questions/${encodePathSegment(domain)}/${encodePathSegment(questionId)}/private`,
        {
        method: "POST",
        }
      ),

    setPublic: (domain: string, questionId: number) =>
      request<SetQuestionPrivateResponse>(
        `/api/v2/questions/${encodePathSegment(domain)}/${encodePathSegment(questionId)}/public`,
        {
        method: "POST",
        }
      ),
  },

  user: {
    updateProfile: (data: UpdateProfileRequest) =>
      request<UpdateProfileResponse>("/api/v2/user/profile", {
        method: "POST",
        body: JSON.stringify(data),
      }),

    uploadAvatar: (file: File) => {
      const formData = new FormData()
      formData.append("file", file)
      return request<UploadUserAssetResponse>("/api/v2/user/avatar", {
        method: "POST",
        body: formData,
      })
    },

    uploadBackground: (file: File) => {
      const formData = new FormData()
      formData.append("file", file)
      return request<UploadUserAssetResponse>("/api/v2/user/background", {
        method: "POST",
        body: formData,
      })
    },

    updateHarassment: (data: UpdateHarassmentRequest) =>
      request<UpdateHarassmentResponse>("/api/v2/user/harassment", {
        method: "POST",
        body: JSON.stringify(data),
      }),

    questions: {
      list: (params?: { page_size?: number; cursor?: string }) => {
        const searchParams = new URLSearchParams()
        if (params?.page_size) {
          searchParams.set("page_size", params.page_size.toString())
        }
        if (params?.cursor) {
          searchParams.set("cursor", params.cursor)
        }

        const query = searchParams.toString()
        return request<GetQuestionsResponse>(`/api/v2/user/questions${query ? `?${query}` : ""}`)
      },

      stats: () => request<QuestionStatsResponse>("/api/v2/user/questions/stats"),

      markAllViewed: () =>
        request<MarkAllQuestionsViewedResponse>("/api/v2/user/questions/viewed", {
          method: "POST",
        }),

      markViewed: (questionId: number) =>
        request<MarkQuestionViewedResponse>(`/api/v2/user/questions/${encodePathSegment(questionId)}/viewed`, {
          method: "POST",
        }),
    },

    exportData: () => request<ExportDataResponse>("/api/v2/user/export"),

    deactivate: () =>
      request<DeactivateResponse>("/api/v2/user/deactivate", {
        method: "POST",
      }),
  },
}
