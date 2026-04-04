"use client";

import { GoogleOAuthProvider, GoogleLogin } from "@react-oauth/google";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { API_URL } from "../../constants";
import { useSession } from "../../hooks/useSession";
import type { SessionUser } from "../../lib/session";

function LoginForm({ googleDisabled }: { googleDisabled: boolean }) {
  const router = useRouter();
  const params = useSearchParams();
  const next = params.get("next") || "/";
  const { login } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [resetEmail, setResetEmail] = useState("");
  const [resetMsg, setResetMsg] = useState("");

  const redirectForRole = (user: SessionUser) => {
    if (user.role === "business") {
      router.push("/dashboard");
      return;
    }
    if (user.role === "admin") {
      router.push("/admin");
      return;
    }
    router.push(next === "/login" ? "/" : next);
  };

  const handleLocalLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setLoading(true);
    try {
      const res = await fetch(`${API_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const body = await res.json();
      if (!res.ok) {
        setErr(body?.error?.message || "Login failed");
        return;
      }
      const token = body.data?.token as string;
      const user = body.data?.user as SessionUser;
      if (!token || !user) {
        setErr("Invalid response");
        return;
      }
      login(token, user);
      redirectForRole(user);
    } catch {
      setErr("Network error");
    } finally {
      setLoading(false);
    }
  };

  const handleGoogle = async (credential: string | undefined) => {
    if (!credential) {
      setErr("Google sign-in failed");
      return;
    }
    setErr("");
    setLoading(true);
    try {
      const res = await fetch(`${API_URL}/api/auth/google`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ idToken: credential }),
      });
      const body = await res.json();
      if (!res.ok) {
        setErr(body?.error?.message || "Google verify failed");
        return;
      }
      const token = body.data?.token as string;
      const user = body.data?.user as SessionUser;
      if (!token || !user) {
        setErr("Invalid response");
        return;
      }
      login(token, user);
      redirectForRole(user);
    } catch {
      setErr("Network error");
    } finally {
      setLoading(false);
    }
  };

  const handleForgot = async (e: React.FormEvent) => {
    e.preventDefault();
    setResetMsg("");
    try {
      await fetch(`${API_URL}/api/auth/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: resetEmail }),
      });
      setResetMsg("If the account exists, check server logs for the reset link (simulated email).");
    } catch {
      setResetMsg("Request failed");
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-lg border border-slate-200 p-8 space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">RideSync</h1>
          <p className="text-slate-600 text-sm mt-1">Sign in to continue</p>
        </div>

        {!googleDisabled && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">Riders & drivers</p>
            <div className="flex justify-center">
              <GoogleLogin
                onSuccess={(c) => handleGoogle(c.credential)}
                onError={() => setErr("Google popup error")}
                useOneTap={false}
              />
            </div>
          </div>
        )}

        {googleDisabled && (
          <p className="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
            Set <code className="text-xs">NEXT_PUBLIC_GOOGLE_CLIENT_ID</code> for Google sign-in.
          </p>
        )}

        <div className="border-t border-slate-200 pt-6 space-y-3">
          <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">Admin & business</p>
          <form onSubmit={handleLocalLogin} className="space-y-3">
            <Input
              type="email"
              placeholder="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
            />
            <Input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
            {err && <p className="text-sm text-red-600">{err}</p>}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Signing in…" : "Sign in with email"}
            </Button>
          </form>
        </div>

        <div className="border-t border-slate-200 pt-4 space-y-2">
          <p className="text-xs text-slate-500">Business — forgot password</p>
          <form onSubmit={handleForgot} className="flex gap-2">
            <Input
              type="email"
              placeholder="Account email"
              value={resetEmail}
              onChange={(e) => setResetEmail(e.target.value)}
            />
            <Button type="submit" variant="outline">
              Send
            </Button>
          </form>
          {resetMsg && <p className="text-xs text-slate-600">{resetMsg}</p>}
        </div>

        <p className="text-center text-sm text-slate-500">
          <Link href="/" className="text-blue-600 hover:underline">
            Back to home
          </Link>
        </p>
      </div>
    </div>
  );
}

function LoginPageInner() {
  const cid = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";
  if (cid) {
    return (
      <GoogleOAuthProvider clientId={cid}>
        <LoginForm googleDisabled={false} />
      </GoogleOAuthProvider>
    );
  }
  return <LoginForm googleDisabled />;
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center text-slate-500">Loading…</div>
      }
    >
      <LoginPageInner />
    </Suspense>
  );
}
