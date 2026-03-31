export interface User {
  id: number
  name: string
  email: string
  avatar: string
  domain: string
  background: string
  intro: string
  notify: string
  harassment_setting: string
  block_words?: string
  routing_workspace_id?: number
  created_at: string
  updated_at: string
}

export interface Question {
  id: number
  user_id: number
  tenant_id?: number
  workspace_id?: number
  content: string
  answer: string
  status: string
  assigned_to_user_id?: number | null
  internal_note?: string
  is_private: boolean
  viewed_at?: string | null
  resolved_at?: string | null
  created_at: string
  updated_at: string
}

export interface PublicUser {
  name: string
  avatar: string
  domain: string
  background: string
  intro: string
}

export interface TenantSummary {
  uid: string
  name: string
  plan: string
  role: string
  is_personal: boolean
  created_at: string
}

export interface WorkspaceSummary {
  id: number
  uid: string
  name: string
  description?: string
  is_default: boolean
  created_at: string
  tenant: TenantSummary
}

export interface TenantMemberSummary {
  user_id: number
  name: string
  email: string
  domain: string
  role: string
  joined_at: string
}

export interface ApiResponse<T = any> {
  success: boolean
  message?: string
  data?: T
}

export interface LoginRequest {
  email: string
  password: string
  recaptcha: string
}

export interface LoginResponse {
  success: boolean
  message?: string
  user?: User
  token?: string
}

export interface RegisterRequest {
  name: string
  email: string
  password: string
  domain: string
  recaptcha: string
}

export interface RegisterResponse {
  success: boolean
  message?: string
  user?: User
  token?: string
}

export interface CreateQuestionRequest {
  content: string
  is_private: boolean
  receive_reply_email: string
  recaptcha: string
}

export interface CreateQuestionResponse {
  success: boolean
  message?: string
  question?: Question
}

export interface GetQuestionsResponse {
  success: boolean
  questions: Question[]
  next_cursor?: string
}

export interface GetUserResponse {
  success: boolean
  user: PublicUser
}

export interface ListTenantsResponse {
  success: boolean
  tenants: TenantSummary[]
}

export interface ListTenantMembersResponse {
  success: boolean
  tenant: TenantSummary
  members: TenantMemberSummary[]
}

export interface AddTenantMemberRequest {
  email: string
  role: string
}

export interface AddTenantMemberResponse {
  success: boolean
  message?: string
  member: TenantMemberSummary
}

export interface UpdateTenantMemberRoleRequest {
  role: string
}

export interface UpdateTenantMemberRoleResponse {
  success: boolean
  message?: string
  member: TenantMemberSummary
}

export interface RemoveTenantMemberResponse {
  success: boolean
  message?: string
}

export interface ListWorkspacesResponse {
  success: boolean
  workspaces: WorkspaceSummary[]
}

export interface CreateWorkspaceRequest {
  tenant_uid: string
  name: string
  description: string
}

export interface CreateWorkspaceResponse {
  success: boolean
  message?: string
  workspace: WorkspaceSummary
}

export interface GetQuestionResponse {
  success: boolean
  question?: Question
  can_delete?: boolean
}

export interface ListWorkspaceQuestionsResponse {
  success: boolean
  workspace: WorkspaceSummary
  questions: Question[]
  next_cursor?: string
}

export interface WorkspaceQuestionStats {
  total_count: number
  new_count: number
  in_progress_count: number
  answered_count: number
  closed_count: number
  private_count: number
  assigned_count: number
  unassigned_count: number
  resolved_count: number
  unresolved_count: number
}

export interface GetWorkspaceQuestionStatsResponse {
  success: boolean
  workspace: WorkspaceSummary
  stats: WorkspaceQuestionStats
}

export interface WorkspaceQuestionMutationResponse {
  success: boolean
  message?: string
  question?: Question
}

export interface UpdateWorkspaceQuestionStatusRequest {
  status: string
}

export interface UpdateWorkspaceQuestionAssigneeRequest {
  assigned_to_user_id?: number | null
}

export interface UpdateWorkspaceQuestionInternalNoteRequest {
  internal_note: string
}

export interface UpdateWorkspaceQuestionPrivacyRequest {
  is_private: boolean
}

export interface SetWorkspaceIntakeResponse {
  success: boolean
  message?: string
  user?: User
  workspace: WorkspaceSummary
}

export interface AnswerQuestionRequest {
  answer: string
}

export interface AnswerQuestionResponse {
  success: boolean
  message?: string
}

export interface DeleteQuestionResponse {
  success: boolean
  message?: string
}

export interface SetQuestionPrivateResponse {
  success: boolean
  message?: string
}

export interface UpdateProfileRequest {
  name: string
  intro: string
  old_password: string
  new_password: string
  notify_email: boolean
}

export interface UpdateProfileResponse {
  success: boolean
  message?: string
  user?: User
}

export interface UploadUserAssetResponse {
  success: boolean
  message?: string
  url?: string
  user?: User
}

export interface UpdateHarassmentRequest {
  register_only: boolean
  block_words: string
}

export interface UpdateHarassmentResponse {
  success: boolean
  message?: string
  user?: User
}

export interface ExportDataResponse {
  success: boolean
  user?: User
  questions?: Question[]
}

export interface DeactivateResponse {
  success: boolean
}

export interface QuestionStatsResponse {
  success: boolean
  total_count: number
  answered_count: number
  unread_count: number
  pending_count: number
}

export interface MarkQuestionViewedResponse {
  success: boolean
  viewed_at?: string
}

export interface MarkAllQuestionsViewedResponse {
  success: boolean
  viewed_at?: string
  viewed_count: number
}
