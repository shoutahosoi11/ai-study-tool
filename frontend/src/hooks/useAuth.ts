import { useEffect, useState } from "react";
import type { User } from "firebase/auth";
import { onAuthChanged } from "../api/auth";

type AuthState = {
  user: User | null;
  loading: boolean;
};

export function useAuth(): AuthState {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(function () {
    const unsubscribe = onAuthChanged(function (u) {
      setUser(u);
      setLoading(false);
    });
    return unsubscribe;
  }, []);

  return { user, loading };
}
