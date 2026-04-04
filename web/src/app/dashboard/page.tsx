"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "../../components/ui/button";
import { apiFetch } from "../../lib/api";
import { useSession } from "../../hooks/useSession";

export default function DashboardPage() {
  const { session, ready, logout } = useSession();
  const [revenue, setRevenue] = useState<unknown>(null);
  const [regions, setRegions] = useState<unknown>(null);
  const [categories, setCategories] = useState<unknown>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!ready || !session) return;
    if (session.user.role !== "business" && session.user.role !== "admin") return;

    (async () => {
      try {
        const [r1, r2, r3] = await Promise.all([
          apiFetch("/api/finance/dashboard/revenue"),
          apiFetch("/api/finance/dashboard/regions"),
          apiFetch("/api/finance/dashboard/categories"),
        ]);
        const b1 = await r1.json();
        const b2 = await r2.json();
        const b3 = await r3.json();
        if (!r1.ok) setError(b1?.error?.message || "Revenue failed");
        else setRevenue(b1.data);
        if (r2.ok) setRegions(b2.data);
        if (r3.ok) setCategories(b3.data);
      } catch {
        setError("Network error");
      }
    })();
  }, [ready, session]);

  if (!ready) {
    return (
      <div className="min-h-screen flex items-center justify-center text-slate-500">Loading…</div>
    );
  }

  if (!session) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-slate-600">Sign in with a business account.</p>
        <Button asChild>
          <Link href="/login?next=/dashboard">Sign in</Link>
        </Button>
      </div>
    );
  }

  if (session.user.role !== "business" && session.user.role !== "admin") {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-slate-600">Dashboard is for business or admin roles.</p>
        <Button asChild variant="outline">
          <Link href="/">Home</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="max-w-5xl mx-auto space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Finance dashboard</h1>
            <p className="text-sm text-slate-500">{session.user.email} · {session.user.role}</p>
          </div>
          <div className="flex gap-2">
            {session.user.role === "admin" && (
              <Button variant="outline" asChild>
                <Link href="/admin">Admin</Link>
              </Button>
            )}
            <Button variant="outline" asChild>
              <Link href="/">Home</Link>
            </Button>
            <Button variant="ghost" onClick={() => { logout(); window.location.href = "/"; }}>
              Sign out
            </Button>
          </div>
        </div>

        {error && <p className="text-red-600 text-sm">{error}</p>}

        <div className="grid gap-6 md:grid-cols-1">
          <section className="bg-white rounded-xl border border-slate-200 p-4">
            <h2 className="font-medium text-slate-800 mb-2">Global revenue</h2>
            <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-64">
              {JSON.stringify(revenue, null, 2)}
            </pre>
          </section>
          <section className="bg-white rounded-xl border border-slate-200 p-4">
            <h2 className="font-medium text-slate-800 mb-2">Regional analytics</h2>
            <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-64">
              {JSON.stringify(regions, null, 2)}
            </pre>
          </section>
          <section className="bg-white rounded-xl border border-slate-200 p-4">
            <h2 className="font-medium text-slate-800 mb-2">Category insights</h2>
            <pre className="text-xs bg-slate-50 p-3 rounded-lg overflow-auto max-h-64">
              {JSON.stringify(categories, null, 2)}
            </pre>
          </section>
        </div>
      </div>
    </div>
  );
}
