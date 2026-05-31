import { BrowserRouter, Navigate, Route, Routes, useSearchParams } from "react-router-dom";
import { ProtectedRoute } from "./components/common/ProtectedRoute";
import { AppShell } from "./components/layout/AppShell";
import { OnboardingGuide } from "./components/system/OnboardingGuide";
import {
  AdminAdMobPage,
  AdminBillingPage,
  AdminJobsPage,
  AdminLLMPage,
  AdminOverviewPage,
  AdminUserDetailPage,
  AdminUsersPage,
} from "./pages/admin/AdminPages";
import { LoginPage } from "./pages/auth/LoginPage";
import { SignupPage } from "./pages/auth/SignupPage";
import { ExtensionConnectPage } from "./pages/extension/ExtensionConnectPage";
import { ProfilePage } from "./pages/profile/ProfilePage";
import { QuestionPage } from "./pages/question/QuestionPage";
import { TimelinePage } from "./pages/timeline/TimelinePage";

function MainPage() {
  const [searchParams] = useSearchParams();
  const tab = searchParams.get("tab") ?? "timeline";

  return (
    <AppShell>
      <OnboardingGuide />
      {tab === "timeline" && <TimelinePage />}
      {tab === "question" && <QuestionPage />}
      {tab === "profile" && <ProfilePage />}
    </AppShell>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/signup" element={<SignupPage />} />
        <Route
          path="/extension/connect"
          element={
            <ProtectedRoute>
              <ExtensionConnectPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin"
          element={
            <ProtectedRoute>
              <AdminOverviewPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/users"
          element={
            <ProtectedRoute>
              <AdminUsersPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/users/:id"
          element={
            <ProtectedRoute>
              <AdminUserDetailPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/llm"
          element={
            <ProtectedRoute>
              <AdminLLMPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/jobs"
          element={
            <ProtectedRoute>
              <AdminJobsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/billing"
          element={
            <ProtectedRoute>
              <AdminBillingPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/admob"
          element={
            <ProtectedRoute>
              <AdminAdMobPage />
            </ProtectedRoute>
          }
        />
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
