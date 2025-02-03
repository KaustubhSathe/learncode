"use client"

import { useState, useEffect } from "react"
import Editor from "@monaco-editor/react"
import { ChevronDown, ChevronUp, Play } from "lucide-react"
import { useRouter } from 'next/navigation'
import { Problem, User } from '@/types'

const defaultCode = "// Write your solution here\n"

export default function ProblemPage({ params }: { params: { id: string } }) {
  const router = useRouter()
  const [problem, setProblem] = useState<Problem | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [code, setCode] = useState(defaultCode)
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(true)
  const [output, setOutput] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [testInput, setTestInput] = useState("")
  const [expectedOutput, setExpectedOutput] = useState("")
  const [language, setLanguage] = useState('nodejs')

  useEffect(() => {
    const fetchProblem = async () => {
      try {
        const token = localStorage.getItem('auth_token')
        console.log('Problem page - Token:', token ? 'present' : 'missing')
        
        if (!token) {
          router.push('/')
          return
        }

        const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/problems/${params.id}`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        })
        console.log('Response status:', response.status)

        if (!response.ok) {
          const errorText = await response.text()
          console.log('Error response:', errorText)
          throw new Error('Failed to fetch problem')
        }

        const data = await response.json()
        console.log('Problem data:', data)
        
        setProblem(data.problem)
        setUser(data.user)
        setTestInput(data.problem.input)
        setExpectedOutput(data.problem.output)
      } catch (err) {
        console.error('Fetch error:', err)
        setError(err instanceof Error ? err.message : 'An error occurred')
      } finally {
        setLoading(false)
      }
    }

    fetchProblem()
  }, [params.id, router])

  const handleSubmit = async () => {
    try {
      setIsSubmitting(true)
      const token = localStorage.getItem('auth_token')
      console.log('Submit - Token:', token ? 'present' : 'missing')
      if (!token) {
        throw new Error('Not authenticated')
      }

      const submitData = {
        problem_id: params.id,
        code,
        language,
      }
      console.log('Submit - Request data:', submitData)

      // Submit to our backend API
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/submit`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(submitData),
      })
      console.log('Submit - Response status:', response.status)

      if (!response.ok) {
        const errorText = await response.text()
        console.error('Submit - Error response:', errorText)
        throw new Error('Failed to submit code')
      }

      const submission = await response.json()
      console.log('Submit - Submission created:', submission)
    } catch (error) {
      console.error('Submit - Caught error:', error)
      setOutput(error instanceof Error ? error.message : 'Failed to submit code')
    } finally {
      setIsSubmitting(false)
      console.log('Submit - Completed')
    }
  }

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>
  if (!problem) return <div>Problem not found</div>

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
      
      <div className="mb-8">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-3xl font-bold">{problem.title}</h1>
          <span className={`px-3 py-1 rounded-full text-sm ${
            problem.difficulty === 'Easy' ? 'bg-green-100 text-green-800' :
            problem.difficulty === 'Medium' ? 'bg-yellow-100 text-yellow-800' :
            'bg-red-100 text-red-800'
          }`}>
            {problem.difficulty}
          </span>
        </div>
        
        <div className="prose max-w-none">
          <h3 className="text-xl font-semibold mb-2">Description</h3>
          <p className="mb-4">{problem.description}</p>
          
          <h3 className="text-xl font-semibold mb-2">Example Input</h3>
          <pre className="bg-gray-100 p-4 rounded-lg mb-4">{problem.input}</pre>
          
          <h3 className="text-xl font-semibold mb-2">Expected Output</h3>
          <pre className="bg-gray-100 p-4 rounded-lg">{problem.output}</pre>
        </div>
      </div>

      {/* Code Editor and Output */}
      <div className="flex-1 grid grid-cols-2 gap-4 p-4">
        <div className="flex flex-col space-y-4">
          <div className="flex items-center justify-between">
            <select 
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              className="bg-background border rounded-md px-2 py-1"
            >
              <option value="nodejs">Node.js</option>
              <option value="python">Python</option>
              <option value="java">Java</option>
            </select>
            <button
              onClick={handleSubmit}
              className="flex items-center space-x-2 bg-primary text-primary-foreground px-4 py-2 rounded-md hover:bg-primary/90"
            >
              <Play className="w-4 h-4" />
              <span>Run</span>
            </button>
          </div>
          <div className="flex-1 border rounded-md overflow-hidden">
            <Editor
              height="100%"
              defaultLanguage={language}
              theme="vs-dark"
              value={code}
              onChange={(value) => setCode(value || "")}
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
              }}
            />
          </div>
        </div>

        <div className="flex flex-col space-y-4">
          <h2 className="font-medium">Test Cases</h2>
          <div className="flex-1 border rounded-md p-4 space-y-4">
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Input:</h3>
              <pre className="bg-muted p-2 rounded-md text-sm">
                {problem.input}
              </pre>
            </div>
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Expected Output:</h3>
              <pre className="bg-muted p-2 rounded-md text-sm">
                {problem.output}
              </pre>
            </div>
            {output && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium">Output:</h3>
                <pre className="bg-muted p-2 rounded-md text-sm">{output}</pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
} 