import { apiClient } from "./client";
import type { AnswerResult, Question } from "../types/question";

export async function listQuestions(): Promise<Question[]> {
  const res = await apiClient.get<{ data: Question[] }>("/questions");
  return res.data.data ?? [];
}

export async function submitAnswer(questionId: string, userAnswer: string): Promise<AnswerResult> {
  const res = await apiClient.post<{ data: AnswerResult }>(`/questions/${questionId}/answer`, {
    user_answer: userAnswer,
  });
  return res.data.data;
}
