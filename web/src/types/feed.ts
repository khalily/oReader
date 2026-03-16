// Feed types
export interface Feed {
  id: string
  title: string
  feed_url: string
  description: string | null
  image_url: string | null
  last_fetched_at: string | null
  created_at: string
}

// Feed with user-specific data
export interface UserFeed extends Feed {
  unread_count: number
  position?: number
}

// Item types
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

// Article with user state and feed info
export interface Article extends Item {
  feed: Feed
  user_state: UserItemState | null
}

// Request types
export interface CreateFeedRequest {
  feed_url: string
}

export interface CreateFeedResponse {
  feed: Feed
  new_item_count: number
}

export interface ListFeedsResponse {
  feeds: UserFeed[]
  total: number
}

export interface GetFeedResponse {
  feed: Feed
  item_count: number
}

export interface RefreshFeedResponse {
  updated: boolean
  new_item_count: number
}

export interface ListItemsOptions {
  limit?: number
  cursor?: string
  feed_id?: string
  starred?: boolean
  read?: boolean
}

export interface ListItemsResponse {
  items: Article[]
  total: number
  has_more: boolean
  next_cursor?: string
}

export interface ToggleStarRequest {
  starred: boolean
}

export interface ToggleReadRequest {
  read: boolean
}

export interface MarkAllReadResponse {
  count: number
}

// OPML types
export interface OpmlImportResponse {
  job_id: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  total_feeds: number
  processed_feeds: number
  failed_feeds: number
}
