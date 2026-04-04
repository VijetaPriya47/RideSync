"use client";

import Link from "next/link";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { API_URL } from "../../constants";

function ResetForm() {
  const params = useSearchParams();
  const tokenFromQuery = params.get("token") || "";
  const [token, setToken] = useState(tokenFromQuery);
  const [password, setPassword] = useState("");
  const [msg, setMsg] = useState("");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg("");
    const res = await fetch(`${API_URL}/api/auth/reset-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, newPassword: password }),
    });
    const body = await res.json();
    if (!res.ok) {
      setMsg(body?.error?.message || "Reset failed");
      return;
    }
    setMsg("Password updated. You can sign in.");
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 p-4">
      <form onSubmit={submit} className="bg-white rounded-xl border border-slate-200 p-8 w-full max-w-md space-y-4">
        <h1 className="text-xl font-semibold">Reset password</h1>
        <Input placeholder="Reset token" value={token} onChange={(e) => setToken(e.target.value)} />
        <Input type="password" placeholder="New password" value={password} onChange={(e) => setPassword(e.target.value)} />
        {msg && <p className="text-sm text-slate-600">{msg}</p>}
        <Button type="submit" className="w-full">Update password</Button>
        <p className="text-center text-sm">
          <Link href="/login" className="text-blue-600 hover:underline">Back to login</Link>
        </p>
      </form>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center">Loading…</div>}>
      <ResetForm />
    </Suspense>
  );
}
