import { BrowserRouter, Navigate, Route, Routes, useSearchParams } from "react-router-dom";
import { ProtectedRoute } from "./components/common/ProtectedRoute";
import { AppLayout } from "./components/layout/AppLayout";
import { LoginPage } from "./pages/auth/LoginPage";
import { SignupPage } from "./pages/auth/SignupPage";
import { ProfilePage } from "./pages/profile/ProfilePage";
import { QuestionPage } from "./pages/question/QuestionPage";
import { TimelinePage } from "./pages/timeline/TimelinePage";

function MainPage() {
  const [searchParams] = useSearchParams();
  const tab = searchParams.get("tab") ?? "timeline";

  return (
    <AppLayout>
      {tab === "timeline" && <TimelinePage />}
      {tab === "question" && <QuestionPage />}
      {tab === "profile" && <ProfilePage />}
    </AppLayout>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/signup" element={<SignupPage />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <MainPage />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
