import { useState } from "react";
import { listIncorrectQuestions, listSavedQuestions } from "../../api/questions";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { IncorrectQuestion, Question, SavedQuestion } from "../../types/question";
import { IncorrectQuestionsModal } from "../profile/IncorrectQuestionsModal";
import { SavedQuestionsModal } from "../profile/SavedQuestionsModal";
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
  const [savedQuestionsOpen, setSavedQuestionsOpen] = useState(false);
  const [savedQuestionsList, setSavedQuestionsList] = useState<SavedQuestion[]>([]);
  const [savedQuizQuestions, setSavedQuizQuestions] = useState<Question[]>([]);
  const [savedQuizNotes, setSavedQuizNotes] = useState<Record<string, string>>({});
  const [incorrectQuestionsLoading, setIncorrectQuestionsLoading] = useState(false);
  const [incorrectQuestionsError, setIncorrectQuestionsError] = useState("");
  const [incorrectQuestionsOpen, setIncorrectQuestionsOpen] = useState(false);
  const [incorrectQuestionsList, setIncorrectQuestionsList] = useState<IncorrectQuestion[]>([]);
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

      setSavedQuestionsList(savedQuestions);
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

      setIncorrectQuestionsList(incorrectQuestions);
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

  async function handleOpenSavedQuestions() {
    setSavedQuestionsOpen(true);
    setSavedQuestionsError("");
    setSavedQuestionsLoading(true);
    try {
      const savedQuestions = await listSavedQuestions();
      setSavedQuestionsList(savedQuestions);
      setSavedQuizNotes(
        savedQuestions.reduce(function (acc, savedQuestion) {
          acc[savedQuestion.id] = savedQuestion.note ?? "";
          return acc;
        }, {} as Record<string, string>)
      );
      setSavedQuizQuestions(savedQuestions.map(toQuestion));
      if (savedQuestions.length === 0) {
        setSavedQuestionsError("保存された問題がない");
      }
    } catch {
      setSavedQuestionsError("保存済み問題の取得に失敗しました");
    } finally {
      setSavedQuestionsLoading(false);
    }
  }

  async function handleOpenIncorrectQuestions() {
    setIncorrectQuestionsOpen(true);
    setIncorrectQuestionsError("");
    setIncorrectQuestionsLoading(true);
    try {
      const incorrectQuestions = await listIncorrectQuestions();
      setIncorrectQuestionsList(incorrectQuestions);
      setIncorrectQuizNotes(
        incorrectQuestions.reduce(function (acc, incorrectQuestion) {
          acc[incorrectQuestion.id] = incorrectQuestion.note ?? "";
          return acc;
        }, {} as Record<string, string>)
      );
      setIncorrectQuizQuestions(incorrectQuestions.map(toQuestion));
      if (incorrectQuestions.length === 0) {
        setIncorrectQuestionsError("間違った問題がない");
      }
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
          border: `1px solid ${theme.colors.border}`,
          borderRadius: theme.radius.md,
          background: theme.colors.background,
          padding: theme.spacing.md,
          display: "flex",
          flexDirection: "column",
          gap: theme.spacing.sm,
          marginBottom: theme.spacing.md,
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: theme.spacing.md }}>
          <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: 0 }}>問題</h2>
          <span
            style={{
              border: `1px solid ${theme.colors.border}`,
              borderRadius: theme.radius.full,
              padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
              color: theme.colors.secondary,
              fontSize: theme.fontSize.xs,
              fontWeight: 700,
              whiteSpace: "nowrap",
            }}
          >
            既定の出題数
          </span>
        </div>
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          取り込んだ本ごとに、設定した問題数の分だけそのまま解いていけます。
        </p>
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
      <QuestionCollectionCard
        title="保存済み問題"
        description="解き終わって保存した問題を、もう一度解いたり一覧で見返せます。"
        countLabel={savedQuizQuestions.length > 0 ? `${savedQuizQuestions.length}問` : "復習"}
        solving={savedQuestionsLoading}
        onSolve={handleRetrySavedQuestions}
        onViewList={handleOpenSavedQuestions}
      />
      <QuestionCollectionCard
        title="間違った問題"
        description="直近で間違えた問題を、もう一度解いたり一覧で見返せます。"
        countLabel={incorrectQuizQuestions.length > 0 ? `${incorrectQuizQuestions.length}問` : "復習"}
        solving={incorrectQuestionsLoading}
        onSolve={handleRetryIncorrectQuestions}
        onViewList={handleOpenIncorrectQuestions}
      />
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
      {savedQuestionsOpen && (
        <SavedQuestionsModal
          questions={savedQuestionsList}
          loading={savedQuestionsLoading}
          error={savedQuestionsError}
          onClose={function () {
            setSavedQuestionsOpen(false);
          }}
        />
      )}
      {incorrectQuestionsOpen && (
        <IncorrectQuestionsModal
          questions={incorrectQuestionsList}
          loading={incorrectQuestionsLoading}
          error={incorrectQuestionsError}
          onClose={function () {
            setIncorrectQuestionsOpen(false);
          }}
        />
      )}
    </div>
  );
}

function QuestionCollectionCard({
  title,
  description,
  countLabel,
  solving,
  onSolve,
  onViewList,
}: {
  title: string;
  description: string;
  countLabel: string;
  solving: boolean;
  onSolve: () => void;
  onViewList: () => void;
}) {
  return (
    <div
      style={{
        border: `1px solid ${theme.colors.border}`,
        borderRadius: theme.radius.md,
        background: theme.colors.background,
        padding: theme.spacing.md,
        display: "flex",
        flexDirection: "column",
        gap: theme.spacing.sm,
        marginBottom: theme.spacing.md,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: theme.spacing.md }}>
        <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
          <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>{title}</p>
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>{description}</p>
        </div>
        <span
          style={{
            border: `1px solid ${theme.colors.border}`,
            borderRadius: theme.radius.full,
            padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
            color: theme.colors.secondary,
            fontSize: theme.fontSize.xs,
            fontWeight: 700,
            whiteSpace: "nowrap",
          }}
        >
          {countLabel}
        </span>
      </div>
      <div style={{ display: "flex", gap: theme.spacing.sm }}>
        <div style={{ flex: 1 }}>
          <Button fullWidth onClick={onSolve} loading={solving}>
            問題を解く
          </Button>
        </div>
        <div style={{ flex: 1 }}>
          <Button variant="outline" fullWidth onClick={onViewList} disabled={solving}>
            一覧を見る
          </Button>
        </div>
      </div>
    </div>
  );
}
