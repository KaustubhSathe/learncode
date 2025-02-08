'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAppSelector } from '@/store/hooks'
import ReactMarkdown from 'react-markdown'
import { cn } from "@/lib/utils"
import { useRef } from 'react'

interface ProblemInput {
  title: string
  description: string
  difficulty: 'Easy' | 'Medium' | 'Hard'
  input: string
  output: string
  example_input: string
  example_output: string
}


const initialProblem: ProblemInput = {
  title: '',
  description: '',
  difficulty: 'Easy',
  input: '',
  output: '',
  example_input: '',
  example_output: ''
}


export default function AdminPage() {
  const [problem, setProblem] = useState<ProblemInput>(initialProblem)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [textareaHeight, setTextareaHeight] = useState('300px')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const previewRef = useRef<HTMLDivElement>(null)
  const { user } = useAppSelector(state => state.auth)
  const router = useRouter()

  useEffect(() => {
    const updateHeight = () => {
      if (textareaRef.current) {
        setTextareaHeight(`${textareaRef.current.offsetHeight}px`)
      }
    }

    const resizeObserver = new ResizeObserver(updateHeight)
    if (textareaRef.current) {
      resizeObserver.observe(textareaRef.current)
    }

    return () => resizeObserver.disconnect()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSubmitting(true)
    setError(null)
    const authToken = localStorage.getItem('auth_token')

    try {
      const response = await fetch(`${process.env.API_URL}/admin/add`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`
        },
        body: JSON.stringify(problem)
      })

      if (!response.ok) {
        throw new Error('Failed to create problem')
      }

      setProblem(initialProblem)
      alert('Problem created successfully!')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    } finally {                                                 
      setIsSubmitting(false)
    }
  }

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    setProblem(prev => ({
      ...prev,
      [e.target.name]: e.target.value
    }))
  }

  if (!user || !user.isAdmin) {
    return <div>You are not authorized to access this page</div>
  }

  return (
    user && user.isAdmin &&
    <div className="container mx-auto py-8 px-4">
      <h1 className="text-3xl font-bold mb-8">Add New Problem</h1>
      
      <form onSubmit={handleSubmit} className="w-full space-y-6">
        <div>
          <label className="block text-sm font-medium mb-2">Title</label>
          <input
            type="text"
            name="title"
            value={problem.title}
            onChange={handleChange}
            className="w-full p-2 rounded border dark:border-gray-700 bg-background"
            required
          />
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Description</label>
            <textarea
              ref={textareaRef}
              name="description"
              value={problem.description}
              onChange={handleChange}
              className="w-full p-2 rounded border dark:border-gray-700 bg-background min-h-[300px] font-mono resize-y"
              required
            />
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Preview</label>
            <div 
              ref={previewRef}
              className={cn(
                "p-4 rounded border dark:border-gray-700 bg-background overflow-y-auto",
                "prose dark:prose-invert prose-sm max-w-none",
                "prose-headings:mt-4 prose-headings:mb-2",
                "prose-p:my-2 prose-pre:my-0 prose-pre:bg-muted"
              )}
              style={{ height: textareaHeight }}
            >
              <ReactMarkdown
                components={{
                  pre: ({ node, ...props }) => (
                    <pre className="bg-muted p-2 rounded" {...props} />
                  ),
                  code: ({ node, ...props }) => (
                    <code className="bg-muted px-1 rounded" {...props} />
                  ),
                }}
              >
                {problem.description || '_No content_'}
              </ReactMarkdown>
            </div>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium mb-2">Difficulty</label>
          <select
            name="difficulty"
            value={problem.difficulty}
            onChange={handleChange}
            className="w-full p-2 rounded border dark:border-gray-700 bg-background"
            required
          >
            <option value="Easy">Easy</option>
            <option value="Medium">Medium</option>
            <option value="Hard">Hard</option>
          </select>
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Example Test Input</label>
            <textarea
              name="example_input"
              value={problem.example_input}
              onChange={handleChange}
              rows={3}
              className="w-full p-2 rounded border dark:border-gray-700 bg-background font-mono"
              required

            />
          </div>

          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Example Expected Output</label>
            <textarea
              name="example_output"
              value={problem.example_output}
              onChange={handleChange}
              rows={3}
              className="w-full p-2 rounded border dark:border-gray-700 bg-background font-mono"
              required

            />
          </div>
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Test Input</label>
            <textarea
              name="input"
              value={problem.input}
              onChange={handleChange}
              rows={3}
              className="w-full p-2 rounded border dark:border-gray-700 bg-background font-mono"
              required
            />
          </div>

          <div className="flex-1">
            <label className="block text-sm font-medium mb-2">Expected Output</label>
            <textarea
              name="output"
              value={problem.output}
              onChange={handleChange}
              rows={3}
              className="w-full p-2 rounded border dark:border-gray-700 bg-background font-mono"
              required
            />
          </div>
        </div>

        {error && (
          <div className="text-red-500 text-sm">{error}</div>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
        >
          {isSubmitting ? 'Creating...' : 'Create Problem'}
        </button>
      </form>
    </div>
  )
} 