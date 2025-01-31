import { CacheClient, Configurations, CredentialProvider, TopicClient } from '@gomomento/sdk'
import { supabase } from './supabase'

const QUEUE_NAME = 'code-execution'
const momento = new TopicClient({
  configuration: Configurations.Laptop.v1(),
  credentialProvider: CredentialProvider.fromString(process.env.NEXT_PUBLIC_MOMENTO_AUTH_TOKEN!),
})

export interface CodeSubmission {
  problemId: number
  code: string
  language: string
  userId?: string
}

export async function submitCode(submission: CodeSubmission) {
  try {
    // Create a submission record in Supabase
    const { data: submissionRecord, error: submissionError } = await supabase
      .from('submissions')
      .insert({
        problem_id: submission.problemId,
        code: submission.code,
        language: submission.language,
        status: 'pending',
        user_id: submission.userId,
      })
      .select()
      .single()

    if (submissionError) throw submissionError

    // Add to Momento queue
    await momento.publish(QUEUE_NAME, JSON.stringify({
      ...submission,
      submissionId: submissionRecord.id,
    }))

    return submissionRecord
  } catch (error) {
    console.error('Error submitting code:', error)
    throw error
  }
}

export async function getSubmissionStatus(submissionId: string) {
  const { data, error } = await supabase
    .from('submissions')
    .select('*')
    .eq('id', submissionId)
    .single()

  if (error) throw error
  return data
} 