import { apiClient } from "./client";
import type { MeResponse } from "../types/user";

export async function getMe(): Promise<MeResponse> {
  const res = await apiClient.get<MeResponse>("/users/me");
  return res.data;
}

export async function updateQuestionSettings(defaultQuestionCount: number): Promise<MeResponse> {
  const res = await apiClient.put<MeResponse>("/users/me/question-settings", {
    default_question_count: defaultQuestionCount,
  });
  return res.data;
}
