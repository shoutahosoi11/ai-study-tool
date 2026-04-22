import { useState } from "react";
import { listIncorrectQuestions, listSavedQuestions } from "../../api/questions";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { IncorrectQuestion, Question, SavedQuestion } from "../../types/question";
import { KindleBookSection } from "./KindleBookSection";
import { QuestionQuizSessionModal } from "./QuestionQuizSessionModal";

type ReplayQuestionSource = Pick<
  SavedQuestion | IncorrectQuestion,
  "id" | "question_type" | "content" | "options" | "correct_answer" | "explanation"
>;

function toQuestion(source: ReplayQuestionSource): Question {
  return {
    id: source.id,
    question_type: source.question_type,
    content: source.content,
    options: source.options,
    correct_answer: source.correct_answer,
    explanation: source.explanation,
  };
}

export function QuestionPage() {
  const [savedQuestionsLoading, setSavedQuestionsLoading] = useState(false);
  const [savedQuestionsError, setSavedQuestionsError] = useState("");
  const [savedQuizQuestions, setSavedQuizQuestions] = useState<Question[]>([]);
  const [savedQuizNotes, setSavedQuizNotes] = useState<Record<string, string>>({});
  const [incorrectQuestionsLoading, setIncorrectQuestionsLoading] = useState(false);
  const [incorrectQuestionsError, setIncorrectQuestionsError] = useState("");
  const [incorrectQuizQuestions, setIncorrectQuizQuestions] = useState<Question[]>([]);
  const [incorrectQuizNotes, setIncorrectQuizNotes] = useState<Record<string, string>>({});

  async function handleRetrySavedQuestions() {
    setSavedQuestionsError("");
    setSavedQuestionsLoading(true);
    try {
      const savedQuestions = await listSavedQuestions();
      if (savedQuestions.length === 0) {
        setSavedQuestionsError("保存された問題がない");
        return;
      }

      setSavedQuizQuestions(savedQuestions.map(toQuestion));
      setSavedQuizNotes(
        savedQuestions.reduce(function (acc, savedQuestion) {
          acc[savedQuestion.id] = savedQuestion.note ?? "";
          return acc;
        }, {} as Record<string, string>)
      );
    } catch {
      setSavedQuestionsError("保存済み問題の取得に失敗しました");
    } finally {
      setSavedQuestionsLoading(false);
    }
  }

  async function handleRetryIncorrectQuestions() {
    setIncorrectQuestionsError("");
    setIncorrectQuestionsLoading(true);
    try {
      const incorrectQuestions = await listIncorrectQuestions();
      if (incorrectQuestions.length === 0) {
        setIncorrectQuestionsError("間違った問題がない");
        return;
      }

      setIncorrectQuizQuestions(incorrectQuestions.map(toQuestion));
      setIncorrectQuizNotes(
        incorrectQuestions.reduce(function (acc, incorrectQuestion) {
          acc[incorrectQuestion.id] = incorrectQuestion.note ?? "";
          return acc;
        }, {} as Record<string, string>)
      );
    } catch {
      setIncorrectQuestionsError("間違った問題の取得に失敗しました");
    } finally {
      setIncorrectQuestionsLoading(false);
    }
  }

  return (
    <div style={{ padding: theme.spacing.md }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: theme.spacing.md,
          margin: `0 0 ${theme.spacing.md}`,
        }}
        >
        <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: 0 }}>
          問題
        </h2>
        <div style={{ display: "flex", gap: theme.spacing.sm, flexWrap: "wrap", justifyContent: "flex-end" }}>
          <Button variant="outline" onClick={handleRetryIncorrectQuestions} loading={incorrectQuestionsLoading}>
            間違った問題からもう一度解く
          </Button>
          <Button variant="outline" onClick={handleRetrySavedQuestions} loading={savedQuestionsLoading}>
            保存済み問題からもう一度解く
          </Button>
        </div>
      </div>
      {savedQuestionsError && (
        <p style={{ margin: `0 0 ${theme.spacing.md}`, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
          {savedQuestionsError}
        </p>
      )}
      {incorrectQuestionsError && (
        <p style={{ margin: `0 0 ${theme.spacing.md}`, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
          {incorrectQuestionsError}
        </p>
      )}
      <KindleBookSection />
      {savedQuizQuestions.length > 0 && (
        <QuestionQuizSessionModal
          bookTitle="保存済み問題"
          questions={savedQuizQuestions}
          initialNotes={savedQuizNotes}
          mergeInitialNotesIntoExplanation
          mergedNoteLabel="自分のメモ"
          onClose={function () {
            setSavedQuizQuestions([]);
            setSavedQuizNotes({});
          }}
        />
      )}
      {incorrectQuizQuestions.length > 0 && (
        <QuestionQuizSessionModal
          bookTitle="間違った問題"
          questions={incorrectQuizQuestions}
          initialNotes={incorrectQuizNotes}
          mergeInitialNotesIntoExplanation
          mergedNoteLabel="自分のメモ"
          onClose={function () {
            setIncorrectQuizQuestions([]);
            setIncorrectQuizNotes({});
          }}
        />
      )}
    </div>
  );
}
