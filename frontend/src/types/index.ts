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
  created_at: string
  updated_at: string
}

export interface Question {
  id: number
  user_id: number
  content: string
  answer: string
  is_private: boolean
  viewed_at?: string | null
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

export interface GetQuestionResponse {
  success: boolean
  question?: Question
  can_delete?: boolean
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
