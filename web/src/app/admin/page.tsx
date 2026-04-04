"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { API_URL } from "../../constants";
import { apiFetch } from "../../lib/api";
import { useSession } from "../../hooks/useSession";

export default function AdminPage() {
  const { session, ready, logout } = useSession();
  const [logs, setLogs] = useState<unknown>(null);
  const [logError, setLogError] = useState("");
  const [bizEmail, setBizEmail] = useState("");
  const [bizPass, setBizPass] = useState("");
  const [admEmail, setAdmEmail] = useState("");
  const [admPass, setAdmPass] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    if (!ready || !session || session.user.role !== "admin") return;
    (async () => {
      const res = await apiFetch("/api/admin/system-logs?limit=50");
      const body = await res.json();
      if (!res.ok) {
        setLogError(body?.error?.message || "Failed to load logs");
        return;
      }
      setLogs(body.data);
    })();
  }, [ready, session]);

  const createBiz = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg("");
    const res = await apiFetch("/api/admin/users/business", {
      method: "POST",
      body: JSON.stringify({ email: bizEmail, password: bizPass }),
    });
    const body = await res.json();
    if (!res.ok) {
      setMsg(body?.error?.message || "Failed");
      return;
    }
    setMsg(`Created business user ${bizEmail}`);
    setBizEmail("");
    setBizPass("");
  };

  const createAdm = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg("");
    const res = await apiFetch("/api/admin/users/admin", {
      method: "POST",
      body: JSON.stringify({ email: admEmail, password: admPass }),
    });
    const body = await res.json();
    if (!res.ok) {
      setMsg(body?.error?.message || "Failed");
      return;
    }
    setMsg(`Created admin ${admEmail}`);
    setAdmEmail("");
    setAdmPass("");
  };

  if (!ready) {
    return (
      <div className="min-h-screen flex items-center justify-center text-slate-500">Loading…</div>
    );
  }

  if (!session) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-slate-600">Admin sign-in required.</p>
        <Button asChild>
          <Link href="/login?next=/admin">Sign in</Link>
        </Button>
      </div>
    );
  }

  if (session.user.role !== "admin") {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-slate-600">This area is restricted to administrators.</p>
        <Button asChild variant="outline">
          <Link href="/">Home</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="max-w-5xl mx-auto space-y-8">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Admin</h1>
            <p className="text-sm text-slate-500">{session.user.email}</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" asChild>
              <Link href="/dashboard">Dashboard</Link>
            </Button>
            <Button variant="outline" asChild>
              <Link href="/">Home</Link>
            </Button>
            <Button variant="ghost" onClick={() => { logout(); window.location.href = "/"; }}>
              Sign out
            </Button>
          </div>
        </div>

        {msg && <p className="text-sm text-green-700 bg-green-50 border border-green-200 rounded-lg p-3">{msg}</p>}

        <div className="grid md:grid-cols-2 gap-6">
          <section className="bg-white rounded-xl border border-slate-200 p-4 space-y-3">
            <h2 className="font-medium text-slate-800">Create business user</h2>
            <form onSubmit={createBiz} className="space-y-2">
              <Input type="email" placeholder="Email" value={bizEmail} onChange={(e) => setBizEmail(e.target.value)} required />
              <Input type="password" placeholder="Password" value={bizPass} onChange={(e) => setBizPass(e.target.value)} required />
              <Button type="submit" className="w-full">Create</Button>
            </form>
          </section>
          <section className="bg-white rounded-xl border border-slate-200 p-4 space-y-3">
            <h2 className="font-medium text-slate-800">Create admin user</h2>
            <form onSubmit={createAdm} className="space-y-2">
              <Input type="email" placeholder="Email" value={admEmail} onChange={(e) => setAdmEmail(e.target.value)} required />
              <Input type="password" placeholder="Password" value={admPass} onChange={(e) => setAdmPass(e.target.value)} required />
              <Button type="submit" className="w-full">Create</Button>
            </form>
          </section>
        </div>

        <section className="bg-white rounded-xl border border-slate-200 p-4">
          <h2 className="font-medium text-slate-800 mb-2">System audit logs</h2>
          {logError && <p className="text-red-600 text-sm mb-2">{logError}</p>}
          <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-96">
            {JSON.stringify(logs, null, 2)}
          </pre>
        </section>

        <p className="text-xs text-slate-500">
          Password reset tokens are logged by user-auth-service (simulated email). Gateway: {API_URL}
        </p>
      </div>
    </div>
  );
}
