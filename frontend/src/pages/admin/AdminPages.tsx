import { Home, PencilLine, User } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  adminLogoutAll,
  cancelAdminJob,
  fetchAdminAdMob,
  fetchAdminBilling,
  fetchAdminExtensionTokens,
  fetchAdminJobs,
  fetchAdminLLM,
  fetchAdminOverview,
  fetchAdminUser,
  retryAdminJob,
  revokeAdminExtensionToken,
  revokeAllAdminExtensionTokens,
  searchAdminUsers,
  updateAdminLLMBudget,
  type AdminAdMob,
  type AdminBilling,
  type AdminExtensionToken,
  type AdminGenerationJob,
  type AdminLLMOverview,
  type AdminOverview,
  type AdminUser,
} from "../../api/admin";
import { getApiErrorStatus } from "../../api/errors";
import { Button } from "../../components/common/Button";
import { Card } from "../../components/common/Card";
import { Input } from "../../components/common/Input";
import { Spinner } from "../../components/common/Spinner";
import { theme } from "../../theme";

type LoadState<T> =
  | { status: "loading" }
  | { status: "forbidden" }
  | { status: "error"; message: string }
  | { status: "ready"; data: T };

const navItems = [
  { to: "/admin", label: "Overview", icon: Home },
  { to: "/admin/users", label: "Users", icon: User },
  { to: "/admin/llm", label: "LLM", icon: PencilLine },
  { to: "/admin/jobs", label: "Jobs", icon: PencilLine },
  { to: "/admin/billing", label: "Billing", icon: PencilLine },
  { to: "/admin/admob", label: "AdMob", icon: PencilLine },
];

function AdminShell({ children }: { children: ReactNode }) {
  return (
    <div style={{ maxWidth: 1120, margin: "0 auto", padding: theme.spacing.lg }}>
      <header style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: theme.spacing.md, marginBottom: theme.spacing.lg }}>
        <div>
          <h1 style={{ margin: 0, fontSize: theme.fontSize.xl }}>Admin</h1>
          <p style={{ margin: `${theme.spacing.xs} 0 0`, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            運用状態と安全な管理操作
          </p>
        </div>
      </header>
      <nav style={{ display: "flex", gap: theme.spacing.sm, overflowX: "auto", paddingBottom: theme.spacing.md, marginBottom: theme.spacing.md }}>
        {navItems.map(function ({ to, label, icon: Icon }) {
          return (
            <Link key={to} to={to} style={navLinkStyle}>
              <Icon size={16} />
              {label}
            </Link>
          );
        })}
      </nav>
      {children}
    </div>
  );
}

export function AdminOverviewPage() {
  const [state, setState] = useState<LoadState<AdminOverview>>({ status: "loading" });

  useEffect(function () {
    loadAdminData(fetchAdminOverview, setState);
  }, []);

  return (
    <AdminShell>
      <LoadBoundary state={state}>
        {(overview) => (
          <div style={stackStyle}>
            <div style={metricGridStyle}>
              <MetricCard label="LLM requests" value={`${overview.llm_usage_today.request_count} / ${overview.budget.max_requests || 0}`} />
              <MetricCard label="LLM cost" value={`${overview.budget.used_estimated_cost_yen} / ${overview.budget.max_estimated_cost_yen || 0}円`} />
              <MetricCard label="Queued jobs" value={overview.generation_jobs.queued + overview.generation_jobs.enqueue_failed} />
              <MetricCard label="Failed jobs" value={overview.generation_jobs.failed} />
              <MetricCard label="Extension imports" value={overview.extension_import_count} />
              <MetricCard label="Rate limit 429" value={overview.rate_limit_429_count} />
            </div>
            <Card>
              <SectionTitle title="Recent audit log" />
              <Table
                headers={["action", "target", "created"]}
                rows={overview.recent_audit_logs.map(function (log) {
                  return [log.action, `${log.target_type}${log.target_id ? `:${shortID(log.target_id)}` : ""}`, formatDate(log.created_at)];
                })}
                empty="audit logはまだありません"
              />
            </Card>
          </div>
        )}
      </LoadBoundary>
    </AdminShell>
  );
}

export function AdminUsersPage() {
  const [query, setQuery] = useState("");
  const [state, setState] = useState<LoadState<AdminUser[]>>({ status: "loading" });

  async function load(queryValue = query) {
    await loadAdminData(function () {
      return searchAdminUsers(queryValue);
    }, setState);
  }

  useEffect(function () {
    load("");
  }, []);

  return (
    <AdminShell>
      <div style={stackStyle}>
        <Card>
          <form
            onSubmit={function (event) {
              event.preventDefault();
              load(query);
            }}
            style={{ display: "flex", gap: theme.spacing.sm, alignItems: "flex-end" }}
          >
            <div style={{ flex: 1 }}>
              <Input label="Search" value={query} onChange={setQuery} placeholder="email / user id / Firebase UID / Stripe id" />
            </div>
            <Button type="submit">
              検索
            </Button>
          </form>
        </Card>
        <LoadBoundary state={state}>
          {(users) => (
            <Card>
              <Table
                headers={["user", "plan", "subscription", "tokens", "jobs", "detail"]}
                rows={users.map(function (user) {
                  return [
                    user.email || user.username || shortID(user.id),
                    user.plan,
                    user.subscription_status || "-",
                    String(user.extension_token_count),
                    String(user.recent_jobs_count),
                    <Link to={`/admin/users/${user.id}`} style={linkStyle}>開く</Link>,
                  ];
                })}
                empty="該当ユーザーはありません"
              />
            </Card>
          )}
        </LoadBoundary>
      </div>
    </AdminShell>
  );
}

export function AdminUserDetailPage() {
  const { id = "" } = useParams();
  const [userState, setUserState] = useState<LoadState<AdminUser>>({ status: "loading" });
  const [tokenState, setTokenState] = useState<LoadState<AdminExtensionToken[]>>({ status: "loading" });
  const [message, setMessage] = useState("");

  async function reload() {
    await Promise.all([
      loadAdminData(function () { return fetchAdminUser(id); }, setUserState),
      loadAdminData(function () { return fetchAdminExtensionTokens(id); }, setTokenState),
    ]);
  }

  useEffect(function () {
    reload();
  }, [id]);

  async function runAction(action: () => Promise<unknown>, success: string) {
    setMessage("");
    try {
      await action();
      setMessage(success);
      await reload();
    } catch {
      setMessage("操作に失敗しました。権限または再認証が必要です。");
    }
  }

  return (
    <AdminShell>
      <div style={stackStyle}>
        <LoadBoundary state={userState}>
          {(user) => (
            <Card>
              <SectionTitle title="User detail" />
              <KeyValue rows={[
                ["user id", user.id],
                ["email", user.email || "-"],
                ["firebase uid", user.firebase_uid],
                ["plan", user.plan],
                ["subscription", user.subscription_status || "-"],
                ["question budget", `${user.question_budget.free_used_today} free / ${user.question_budget.available_tokens} tokens`],
                ["last active", user.last_active_at ? formatDate(user.last_active_at) : "-"],
              ]} />
              <div style={{ display: "flex", gap: theme.spacing.sm, marginTop: theme.spacing.md, flexWrap: "wrap" }}>
                <Button variant="outline" onClick={function () { runAction(() => adminLogoutAll(user.id), "全端末ログアウトを実行しました"); }}>
                  Logout all
                </Button>
                <Button variant="outline" onClick={function () { runAction(() => revokeAllAdminExtensionTokens(user.id), "Extension tokenを全てrevokeしました"); }}>
                  Revoke all tokens
                </Button>
              </div>
              {message && <p style={noticeStyle}>{message}</p>}
            </Card>
          )}
        </LoadBoundary>
        <LoadBoundary state={tokenState}>
          {(tokens) => (
            <Card>
              <SectionTitle title="Extension tokens" />
              <Table
                headers={["id", "scopes", "created", "last used", "revoked", "action"]}
                rows={tokens.map(function (token) {
                  return [
                    shortID(token.id),
                    token.scopes.join(", "),
                    formatDate(token.created_at),
                    token.last_used_at ? formatDate(token.last_used_at) : "-",
                    token.revoked_at ? formatDate(token.revoked_at) : "-",
                    token.revoked_at ? "-" : (
                      <Button variant="ghost" onClick={function () { runAction(() => revokeAdminExtensionToken(id, token.id), "Extension tokenをrevokeしました"); }}>
                        Revoke
                      </Button>
                    ),
                  ];
                })}
                empty="Extension tokenはありません"
              />
            </Card>
          )}
        </LoadBoundary>
      </div>
    </AdminShell>
  );
}

export function AdminLLMPage() {
  const [state, setState] = useState<LoadState<AdminLLMOverview>>({ status: "loading" });
  const [maxRequests, setMaxRequests] = useState("");
  const [maxCost, setMaxCost] = useState("");
  const [message, setMessage] = useState("");

  async function reload() {
    await loadAdminData(fetchAdminLLM, function (nextState) {
      setState(nextState);
      if (nextState.status === "ready") {
        setMaxRequests(String(nextState.data.budget.max_requests || 0));
        setMaxCost(String(nextState.data.budget.max_estimated_cost_yen || 0));
      }
    });
  }

  useEffect(function () {
    reload();
  }, []);

  async function saveBudget() {
    setMessage("");
    try {
      await updateAdminLLMBudget(Number(maxRequests), Number(maxCost));
      setMessage("budgetを更新しました");
      await reload();
    } catch {
      setMessage("budget更新に失敗しました。値または再認証を確認してください。");
    }
  }

  return (
    <AdminShell>
      <LoadBoundary state={state}>
        {(llm) => (
          <div style={stackStyle}>
            <div style={metricGridStyle}>
              <MetricCard label="requests" value={`${llm.budget.used_requests} / ${llm.budget.max_requests || 0}`} />
              <MetricCard label="cost" value={`${llm.budget.used_estimated_cost_yen} / ${llm.budget.max_estimated_cost_yen || 0}円`} />
              <MetricCard label="input tokens" value={llm.usage_today.input_tokens} />
              <MetricCard label="output tokens" value={llm.usage_today.output_tokens} />
            </div>
            <Card>
              <SectionTitle title="Budget" />
              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: theme.spacing.md }}>
                <Input label="max requests" type="number" value={maxRequests} onChange={setMaxRequests} />
                <Input label="max cost yen" type="number" value={maxCost} onChange={setMaxCost} />
              </div>
              <div style={{ marginTop: theme.spacing.md }}>
                <Button onClick={saveBudget}>更新</Button>
              </div>
              {message && <p style={noticeStyle}>{message}</p>}
            </Card>
            <Card>
              <SectionTitle title="Provider / model" />
              <Table
                headers={["provider", "model", "requests", "cost"]}
                rows={llm.provider_models.map(function (usage) {
                  return [usage.provider, usage.model, String(usage.request_count), `${usage.estimated_cost_yen}円`];
                })}
                empty="今日の使用量はありません"
              />
            </Card>
            <Card>
              <SectionTitle title="Failed reasons" />
              <Table
                headers={["reason", "count"]}
                rows={llm.failed_job_reasons.map(function (reason) {
                  return [reason.reason, String(reason.count)];
                })}
                empty="今日の失敗jobはありません"
              />
            </Card>
          </div>
        )}
      </LoadBoundary>
    </AdminShell>
  );
}

export function AdminJobsPage() {
  const [status, setStatus] = useState("");
  const [state, setState] = useState<LoadState<AdminGenerationJob[]>>({ status: "loading" });
  const [message, setMessage] = useState("");

  async function reload(statusValue = status) {
    await loadAdminData(function () { return fetchAdminJobs(statusValue); }, setState);
  }

  useEffect(function () {
    reload("");
  }, []);

  async function runJob(action: () => Promise<void>, success: string) {
    setMessage("");
    try {
      await action();
      setMessage(success);
      await reload();
    } catch {
      setMessage("job操作に失敗しました。状態または権限を確認してください。");
    }
  }

  return (
    <AdminShell>
      <div style={stackStyle}>
        <Card>
          <div style={{ display: "flex", gap: theme.spacing.sm, flexWrap: "wrap" }}>
            {["", "queued", "processing", "failed", "enqueue_failed", "completed"].map(function (value) {
              return (
                <Button key={value || "all"} variant={status === value ? "primary" : "outline"} onClick={function () { setStatus(value); reload(value); }}>
                  {value || "all"}
                </Button>
              );
            })}
          </div>
          <p style={noticeStyle}>processing中のjobはworker実行中のため、この画面からはcancelできません。</p>
          {message && <p style={noticeStyle}>{message}</p>}
        </Card>
        <LoadBoundary state={state}>
          {(jobs) => (
            <Card>
              <Table
                headers={["id", "user", "book", "status", "reason", "retry", "updated", "action"]}
                rows={jobs.map(function (job) {
                  const canRetry = job.status === "failed" || job.status === "enqueue_failed";
                  const canCancel = job.status === "queued" || job.status === "enqueue_failed";
                  return [
                    shortID(job.id),
                    shortID(job.user_id),
                    job.book_id,
                    job.status,
                    job.reason,
                    String(job.retry_count),
                    formatDate(job.updated_at),
                    <div style={{ display: "flex", gap: theme.spacing.xs }}>
                      {canRetry && (
                        <Button variant="ghost" onClick={function () { runJob(() => retryAdminJob(job.id), "retryしました"); }}>
                          Retry
                        </Button>
                      )}
                      {canCancel ? (
                        <Button variant="ghost" onClick={function () { runJob(() => cancelAdminJob(job.id), "cancelしました"); }}>
                          Cancel
                        </Button>
                      ) : (
                        <span style={{ color: theme.colors.secondary }}>-</span>
                      )}
                    </div>,
                  ];
                })}
                empty="jobはありません"
              />
            </Card>
          )}
        </LoadBoundary>
      </div>
    </AdminShell>
  );
}

export function AdminBillingPage() {
  const [state, setState] = useState<LoadState<AdminBilling>>({ status: "loading" });
  useEffect(function () {
    loadAdminData(fetchAdminBilling, setState);
  }, []);
  return (
    <AdminShell>
      <LoadBoundary state={state}>
        {(billing) => (
          <Card>
            <SectionTitle title={`Stripe events / failure ${billing.failure_count}`} />
            <Table
              headers={["event id", "type", "processed"]}
              rows={billing.events.map(function (event) {
                return [shortID(event.event_id), event.event_type, formatDate(event.processed_at)];
              })}
              empty="Stripe eventはありません"
            />
          </Card>
        )}
      </LoadBoundary>
    </AdminShell>
  );
}

export function AdminAdMobPage() {
  const [state, setState] = useState<LoadState<AdminAdMob>>({ status: "loading" });
  useEffect(function () {
    loadAdminData(fetchAdminAdMob, setState);
  }, []);
  return (
    <AdminShell>
      <LoadBoundary state={state}>
        {(admob) => (
          <Card>
            <SectionTitle title={`AdMob SSV / duplicate ${admob.duplicate_count}`} />
            <Table
              headers={["transaction", "user", "reward", "verified"]}
              rows={admob.events.map(function (event) {
                return [shortID(event.transaction_id), shortID(event.user_id), String(event.reward_amount), formatDate(event.verified_at)];
              })}
              empty="AdMob SSV eventはありません"
            />
          </Card>
        )}
      </LoadBoundary>
    </AdminShell>
  );
}

async function loadAdminData<T>(loader: () => Promise<T>, setState: (state: LoadState<T>) => void) {
  setState({ status: "loading" });
  try {
    const data = await loader();
    setState({ status: "ready", data });
  } catch (error) {
    if (getApiErrorStatus(error) === 403) {
      setState({ status: "forbidden" });
      return;
    }
    setState({ status: "error", message: "管理データを取得できませんでした" });
  }
}

function LoadBoundary<T>({ state, children }: { state: LoadState<T>; children: (data: T) => ReactNode }) {
  if (state.status === "loading") {
    return <div style={centerStyle}><Spinner /></div>;
  }
  if (state.status === "forbidden") {
    return (
      <div style={forbiddenOverlayStyle}>
        <Card>
          <h2 style={{ margin: 0, fontSize: theme.fontSize.lg }}>403</h2>
          <p style={{ marginBottom: 0, color: theme.colors.secondary }}>管理権限がありません。</p>
        </Card>
      </div>
    );
  }
  if (state.status === "error") {
    return (
      <Card>
        <p style={{ margin: 0, color: theme.colors.danger }}>{state.message}</p>
      </Card>
    );
  }
  return <>{children(state.data)}</>;
}

function MetricCard({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <div style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs, textTransform: "uppercase" }}>{label}</div>
      <div style={{ marginTop: theme.spacing.xs, fontSize: theme.fontSize.xl, fontWeight: 800 }}>{value}</div>
    </Card>
  );
}

function SectionTitle({ title }: { title: string }) {
  return <h2 style={{ margin: `0 0 ${theme.spacing.md}`, fontSize: theme.fontSize.lg }}>{title}</h2>;
}

function Table({ headers, rows, empty }: { headers: string[]; rows: ReactNode[][]; empty: string }) {
  if (rows.length === 0) {
    return <p style={{ margin: 0, color: theme.colors.secondary }}>{empty}</p>;
  }
  return (
    <div style={{ overflowX: "auto" }}>
      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: theme.fontSize.sm }}>
        <thead>
          <tr>
            {headers.map(function (header) {
              return <th key={header} style={thStyle}>{header}</th>;
            })}
          </tr>
        </thead>
        <tbody>
          {rows.map(function (row, rowIndex) {
            return (
              <tr key={rowIndex}>
                {row.map(function (cell, cellIndex) {
                  return <td key={cellIndex} style={tdStyle}>{cell}</td>;
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function KeyValue({ rows }: { rows: Array<[string, ReactNode]> }) {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "minmax(120px, 180px) 1fr", gap: theme.spacing.sm, fontSize: theme.fontSize.sm }}>
      {rows.map(function ([key, value]) {
        return (
          <div key={key} style={{ display: "contents" }}>
            <div style={{ color: theme.colors.secondary }}>{key}</div>
            <div style={{ minWidth: 0, overflowWrap: "anywhere" }}>{value}</div>
          </div>
        );
      })}
    </div>
  );
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function shortID(value: string) {
  if (value.length <= 12) {
    return value;
  }
  return `${value.slice(0, 8)}...${value.slice(-4)}`;
}

const navLinkStyle: CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: theme.spacing.xs,
  color: theme.colors.primary,
  textDecoration: "none",
  border: `1px solid ${theme.colors.border}`,
  borderRadius: theme.radius.sm,
  padding: `${theme.spacing.sm} ${theme.spacing.md}`,
  whiteSpace: "nowrap",
};

const linkStyle: CSSProperties = {
  color: theme.colors.primary,
  fontWeight: 700,
};

const stackStyle: CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: theme.spacing.md,
};

const metricGridStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
  gap: theme.spacing.md,
};

const centerStyle: CSSProperties = {
  display: "flex",
  justifyContent: "center",
  padding: theme.spacing.xl,
};

const forbiddenOverlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 1000,
  background: theme.colors.background,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: theme.spacing.lg,
};

const noticeStyle: CSSProperties = {
  margin: `${theme.spacing.md} 0 0`,
  color: theme.colors.secondary,
  fontSize: theme.fontSize.sm,
};

const thStyle: CSSProperties = {
  textAlign: "left",
  color: theme.colors.secondary,
  borderBottom: `1px solid ${theme.colors.border}`,
  padding: theme.spacing.sm,
  whiteSpace: "nowrap",
};

const tdStyle: CSSProperties = {
  borderBottom: `1px solid ${theme.colors.border}`,
  padding: theme.spacing.sm,
  verticalAlign: "top",
  overflowWrap: "anywhere",
};
