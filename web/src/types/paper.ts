export interface Paper {
  id: string
  user_id: string
  title: string
  authors: string | null       // JSON array string
  abstract: string | null
  keywords: string | null       // JSON array string
  published_year: string | null
  doi: string | null
  pdf_path: string | null
  pdf_size: number
  markdown_content: string | null
  cover_image: string | null
  original_filename: string
  status: PaperStatus
  error: string | null
  created_at: string
  updated_at: string
}

export type PaperStatus = 'pending' | 'processing' | 'completed' | 'failed'

export interface PaperTag {
  id: string
  paper_id: string
  tag: string
}

export interface ListPapersResponse {
  papers: Paper[]
  total: number
}

export interface ListPapersOptions {
  page?: number
  per_page?: number
  q?: string
  year?: string
  tag?: string
  status?: PaperStatus
  sort?: string
  order?: 'asc' | 'desc'
}

export interface PaperStatusResponse {
  id: string
  status: PaperStatus
  progress: number
  error?: string
  created_at: string
  updated_at: string
}

export interface UpdatePaperRequest {
  title?: string
  abstract?: string
  published_year?: string
  doi?: string
}

export interface UpdateTagsRequest {
  tags: string[]
}

export interface UploadPaperResponse {
  id: string
  status: PaperStatus
  original_filename: string
}
