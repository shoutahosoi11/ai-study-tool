export type DefaultQuestionCount = number;

export interface MeResponse {
  id: string;
  username: string;
  display_name: string;
  avatar_url?: string;
  bio?: string;
  university?: string;
  faculty?: string;
  grade?: number;
  country?: string;
  plan: string;
  default_question_count: DefaultQuestionCount;
}
