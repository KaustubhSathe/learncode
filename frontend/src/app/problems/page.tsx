'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

interface Problem {
  id: string
  title: string
  description: string
  difficulty: string
}

interface User {
  id: number
  login: string
}

export default function ProblemsPage() {
  const router = useRouter()
  const [problems, setProblems] = useState<Problem[]>([])
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchProblems = async () => {
      try {
        const token = localStorage.getItem('auth_token')
        console.log('Problems page - Full token:', token)
        console.log('Problems page - Headers:', {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        })
        
        if (!token) {
          router.push('/')
          return
        }

        console.log('Fetching problems from:', process.env.NEXT_PUBLIC_API_URL)
        const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/problems`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        })
        console.log('Response status:', response.status)

        if (!response.ok) {
          const errorText = await response.text()
          console.log('Error response:', errorText)
          throw new Error('Failed to fetch problems')
        }

        const data = await response.json()
        console.log('Response data:', data)
        
        setProblems(data.problems)
        setUser(data.user)
      } catch (err) {
        console.error('Fetch error:', err)
        setError(err instanceof Error ? err.message : 'An error occurred')
      } finally {
        setLoading(false)
      }
    }

    fetchProblems()
  }, [router])


  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          {user && <h2 className="text-xl font-bold">Welcome, {user.login}!</h2>}
        </div>
        <button
          onClick={() => {
            localStorage.removeItem('auth_token')
            router.push('/')
          }}
          className="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
        >
          Logout
        </button>
      </div>
      
      <h1 className="text-3xl font-bold mb-6">Coding Problems</h1>
      
      <div className="grid gap-4">
        {problems.map((problem) => (
          <div 
            key={problem.id} 
            className="border rounded-lg p-4 hover:shadow-lg transition-shadow cursor-pointer"
            onClick={() => {
              console.log('Navigating to problem:', problem.id)
              router.push(`/problems/${problem.id}`)
            }}
          >
            <div className="flex justify-between items-center">
              <h3 className="text-xl font-semibold">{problem.title}</h3>
              <span className={`px-3 py-1 rounded-full text-sm ${
                problem.difficulty === 'Easy' ? 'bg-green-100 text-green-800' :
                problem.difficulty === 'Medium' ? 'bg-yellow-100 text-yellow-800' :
                'bg-red-100 text-red-800'
              }`}>
                {problem.difficulty}
              </span>
            </div>
            <p className="mt-2 text-gray-600">{problem.description}</p>
          </div>
        ))}
      </div>

      {problems.length === 0 && (
        <div className="text-center text-gray-500 mt-8">
          No problems found
        </div>
      )}
    </div>
  )
}