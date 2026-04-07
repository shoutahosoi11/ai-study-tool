export interface Question {
  id: string;
  question_type: "multiple_choice" | "descriptive";
  content: string;
  options: string[];
  correct_answer: string;
  explanation: string;
}

export interface AnswerResult {
  is_correct: boolean;
  correct_answer: string;
  explanation: string;
  score?: number;
  feedback?: string;
}
