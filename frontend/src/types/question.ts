export interface Question {
  id: string;
  question_type: "multiple_choice" | "descriptive";
  content: string;
  options: string[];
  correct_answer: string;
  explanation: string;
}

export interface SaveQuestionResult {
  question_id: string;
  note: string;
  saved: boolean;
}

export interface SavedQuestion {
  id: string;
  question_type: "multiple_choice" | "descriptive";
  content: string;
  options: string[];
  correct_answer: string;
  explanation: string;
  note: string;
  saved_at: string;
}

export interface IncorrectQuestion {
  id: string;
  question_type: "multiple_choice" | "descriptive";
  content: string;
  options: string[];
  correct_answer: string;
  explanation: string;
  note: string;
  answered_at: string;
}

export interface AnswerResult {
  is_correct: boolean;
  correct_answer: string;
  explanation: string;
  score?: number;
  feedback?: string;
}
