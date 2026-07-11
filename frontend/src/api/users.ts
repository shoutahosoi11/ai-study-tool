import { apiClient } from "./client";
import type { MeResponse } from "../types/user";

export async function signUpBackendUser(username: string): Promise<MeResponse> {
  const res = await apiClient.post<MeResponse>("/users/signup", { username });
  return res.data;
}

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

// 事前に reauthenticateForSensitiveAction で再認証しておくこと
// （サーバーは5分以内の再ログインを要求する）。
export async function deleteAccount(): Promise<void> {
  await apiClient.delete("/users/me");
}
