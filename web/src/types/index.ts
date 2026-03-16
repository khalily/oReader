// User types
export interface User {
  id: string
  email: string
  nickname: string | null
  avatar_url: string | null
  auth_provider: 'email' | 'github'
  created_at: string
  updated_at: string
}

// Auth types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  password: string
  nickname?: string
}

export interface AuthResponse {
  user: User
  csrf_token: string
}

// API Error types
export interface ApiError {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

export type ErrorCode =
  | 'VALIDATION_ERROR'
  | 'UNAUTHORIZED'
  | 'TOKEN_EXPIRED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'RATE_LIMIT_EXCEEDED'
  | 'INTERNAL_ERROR'

// Feed types (for reference)
export interface Feed {
  id: string
  title: string
  feed_url: string
  description: string | null
  image_url: string | null
  last_fetched_at: string | null
  created_at: string
}

// Item types (for reference)
export interface Item {
  id: string
  feed_id: string
  guid: string
  title: string
  link: string | null
  description: string | null
  content: string | null
  pub_date: string | null
  creator: string | null
  created_at: string
}

// User item state
export interface UserItemState {
  item_id: string
  is_read: boolean
  is_starred: boolean
  read_at: string | null
}

// Article with user state
export interface Article extends Item {
  feed: Feed
  user_state: UserItemState | null
}
