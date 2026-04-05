export interface Category {
  id: string
  user_id: string
  name: string
  type: 'feed' | 'paper'
  position: number
  created_at: string
}

export interface CreateCategoryRequest {
  name: string
  type: 'feed' | 'paper'
}

export interface RenameCategoryRequest {
  name: string
}

export interface MoveToCategoryRequest {
  category_id: string
}

export interface ListCategoriesResponse {
  categories: Category[]
}
