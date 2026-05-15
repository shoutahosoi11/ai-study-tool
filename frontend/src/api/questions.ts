import { apiClient } from "./client";
import type { AnswerResult, IncorrectQuestion, Question, SaveQuestionResult, SavedQuestion } from "../types/question";

type GenerateQuestionsOptions = {
  questionCount?: number;
  bookTitle?: string;
  bookAuthor?: string;
  customInstruction?: string;
}

export type QuestionStockBook = {
  book_key: string;
  book_title: string;
  book_author: string;
  stock: number;
  target: number;
  preparing: number;
}

export type QuestionStockSyncResponse = {
  books: QuestionStockBook[];
  queued_count: number;
  skipped_due_to_daily_limit: boolean;
}

export async function listQuestions(): Promise<Question[]> {
  const res = await apiClient.get<Question[] | { data?: Question[] }>("/questions");
  return Array.isArray(res.data) ? res.data : res.data.data ?? [];
}

export async function listSavedQuestions(): Promise<SavedQuestion[]> {
  const res = await apiClient.get<SavedQuestion[] | { data?: SavedQuestion[] }>("/questions/saved");
  return Array.isArray(res.data) ? res.data : res.data.data ?? [];
}

export async function listIncorrectQuestions(): Promise<IncorrectQuestion[]> {
  const res = await apiClient.get<IncorrectQuestion[] | { data?: IncorrectQuestion[] }>("/questions/incorrect");
  return Array.isArray(res.data) ? res.data : res.data.data ?? [];
}

export async function submitAnswer(questionId: string, userAnswer: string): Promise<AnswerResult> {
  const res = await apiClient.post<AnswerResult>(`/questions/${questionId}/answer`, {
    user_answer: userAnswer,
  });
  return res.data;
}

export async function saveQuestion(questionId: string, note: string): Promise<SaveQuestionResult> {
  const res = await apiClient.post<SaveQuestionResult>(`/questions/${questionId}/save`, {
    note,
  });
  return res.data;
}

export async function generateQuestions(
  sourceType: string,
  sourceId: string,
  options?: GenerateQuestionsOptions
): Promise<Question[]> {
  const res = await apiClient.get<Question[] | { questions?: Question[] }>("/questions/prepared", {
    params: {
      source_type: sourceType,
      source_id: sourceId,
      question_count: options?.questionCount ?? 0,
      book_title: options?.bookTitle ?? "",
      book_author: options?.bookAuthor ?? "",
    },
  });

  return Array.isArray(res.data) ? res.data : res.data.questions ?? [];
}

export async function syncQuestionStock(): Promise<QuestionStockSyncResponse> {
  const res = await apiClient.post<QuestionStockSyncResponse>("/questions/sync", {});
  return res.data;
}
