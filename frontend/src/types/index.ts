export interface Problem {
  id: string
  title: string
  description: string
  difficulty: 'Easy' | 'Medium' | 'Hard'
  created_at: number  // Unix timestamp
  updated_at: number  // Unix timestamp
  deleted_at?: number // Optional Unix timestamp
  input: string
  output: string
}

export interface User {
  id: number
  login: string
}

export interface ProblemsResponse {
  problems: Problem[]
  user: User
} 