import { apiClient } from './client'

export type Question = {
  id: string
  question_type: 'multiple_choice'
  content: string
  options: string[]
  correct_answer: string
  explanation: string
}

export type SavedQuestion = Question & {
  note: string
  saved_at: string
}

export type IncorrectQuestion = Question & {
  note: string
  answered_at: string
}

export type AnswerResult = {
  is_correct: boolean
  correct_answer: string
  explanation: string
}

export type SaveQuestionResult = {
  question_id: string
  note: string
  saved: boolean
}

export type QuestionStockBook = {
  book_key: string
  book_title: string
  book_author: string
  stock: number
  target: number
  preparing: number
}

export type QuestionStockSyncResponse = {
  books: QuestionStockBook[]
  queued_count: number
  skipped_due_to_daily_limit: boolean
}

type GenerateQuestionsOptions = {
  questionCount?: number
  bookTitle?: string
  bookAuthor?: string
  customInstruction?: string
  highlightStartIndex?: number
  highlightEndIndex?: number
}

export type ManualGenerateQuestionResponse = {
  job_id: string
}

export async function listSavedQuestions(): Promise<SavedQuestion[]> {
  const response = await apiClient.get<SavedQuestion[] | { data?: SavedQuestion[] }>('/questions/saved')
  return Array.isArray(response.data) ? response.data : response.data.data ?? []
}

export async function listIncorrectQuestions(): Promise<IncorrectQuestion[]> {
  const response = await apiClient.get<IncorrectQuestion[] | { data?: IncorrectQuestion[] }>('/questions/incorrect')
  return Array.isArray(response.data) ? response.data : response.data.data ?? []
}

export async function submitAnswer(questionID: string, userAnswer: string): Promise<AnswerResult> {
  const response = await apiClient.post<AnswerResult>(`/questions/${questionID}/answer`, {
    user_answer: userAnswer,
  })
  return response.data
}

export async function saveQuestion(questionID: string, note: string): Promise<SaveQuestionResult> {
  const response = await apiClient.post<SaveQuestionResult>(`/questions/${questionID}/save`, {
    note,
  })
  return response.data
}

export async function generateQuestions(
  sourceType: string,
  sourceID: string,
  options?: GenerateQuestionsOptions
): Promise<Question[]> {
  const response = await apiClient.get<Question[] | { questions?: Question[] }>('/questions/prepared', {
    params: {
      source_type: sourceType,
      source_id: sourceID,
      question_count: options?.questionCount ?? 0,
      book_title: options?.bookTitle ?? '',
      book_author: options?.bookAuthor ?? '',
      highlight_start_index: options?.highlightStartIndex ?? undefined,
      highlight_end_index: options?.highlightEndIndex ?? undefined,
    },
  })

  return Array.isArray(response.data) ? response.data : response.data.questions ?? []
}

export async function syncQuestionStock(): Promise<QuestionStockSyncResponse> {
  const response = await apiClient.post<QuestionStockSyncResponse>('/questions/sync', {})
  return response.data
}

export async function manualGenerateQuestions(bookKey: string, highlightIDs: string[]): Promise<ManualGenerateQuestionResponse> {
  const response = await apiClient.post<ManualGenerateQuestionResponse>('/questions/generate/manual', {
    book_key: bookKey,
    highlight_ids: highlightIDs,
  })
  return response.data
}
