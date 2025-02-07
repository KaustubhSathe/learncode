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
  example_input: string
  example_output: string
}


export interface User {
  id: string
  login: string
  isAdmin: boolean
  created_at: number
  last_login_at: number
}

export interface ProblemsResponse {
  problems: Problem[]
  user: User
} 

export interface Submission {
  id: string
  status: 'pending' | 'running' | 'completed' | 'error'
  result?: string
  type: 'submit' | 'run'
}