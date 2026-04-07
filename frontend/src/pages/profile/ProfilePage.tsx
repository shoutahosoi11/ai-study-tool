import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { signOutUser } from "../../api/auth";
import { apiClient } from "../../api/client";
import { Avatar } from "../../components/common/Avatar";
import { Button } from "../../components/common/Button";
import { Spinner } from "../../components/common/Spinner";
import { theme } from "../../theme";

type UserProfile = {
  id: string;
  username: string;
  display_name: string;
  bio?: string;
  avatar_url?: string;
  follower_count?: number;
  following_count?: number;
};

export function ProfilePage() {
  const navigate = useNavigate();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(function () {
    apiClient.get<{ data: UserProfile }>("/users/me")
      .then(function (res) {
        setProfile(res.data.data);
      })
      .catch(function () {
        setError("プロフィールの取得に失敗しました");
      })
      .finally(function () {
        setLoading(false);
      });
  }, []);

  async function handleSignOut() {
    await signOutUser();
    navigate("/login");
  }

  if (loading) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: theme.spacing.xl }}>
        <Spinner />
      </div>
    );
  }

  if (error) {
    return <p style={{ color: theme.colors.danger, padding: theme.spacing.md }}>{error}</p>;
  }

  return (
    <div style={{ padding: theme.spacing.md }}>
      <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: `0 0 ${theme.spacing.lg}` }}>
        プロフィール
      </h2>
      {profile && (
        <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.lg }}>
          <div style={{ display: "flex", alignItems: "center", gap: theme.spacing.md }}>
            <Avatar name={profile.display_name || profile.username} src={profile.avatar_url} size={64} />
            <div>
              <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.lg }}>{profile.display_name}</p>
              <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>@{profile.username}</p>
            </div>
          </div>
          {profile.bio && (
            <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>{profile.bio}</p>
          )}
          <div style={{ display: "flex", gap: theme.spacing.xl }}>
            <div>
              <span style={{ fontWeight: 700 }}>{profile.following_count ?? 0}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.sm, marginLeft: theme.spacing.xs }}>フォロー</span>
            </div>
            <div>
              <span style={{ fontWeight: 700 }}>{profile.follower_count ?? 0}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.sm, marginLeft: theme.spacing.xs }}>フォロワー</span>
            </div>
          </div>
          <Button variant="outline" onClick={handleSignOut}>
            ログアウト
          </Button>
        </div>
      )}
    </div>
  );
}
