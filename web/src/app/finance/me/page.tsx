"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "../../../components/ui/button";
import { apiFetch } from "../../../lib/api";
import { useSession } from "../../../hooks/useSession";

type Tx = {
  id?: string;
  userId?: string;
  amountCents?: number;
  currency?: string;
  type?: string;
  region?: string;
  status?: string;
  sourceTripId?: string;
  createdAtRfc3339?: string;
};

export default function FinanceMePage() {
  const { session, ready, logout } = useSession();
  const [rows, setRows] = useState<Tx[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!ready || !session || session.user.role !== "customer") return;
    (async () => {
      const res = await apiFetch("/api/finance/me");
      const body = await res.json();
      if (!res.ok) {
        setError(body?.error?.message || "Failed to load");
        return;
      }
      const txs = body.data?.transactions ?? body.data?.Transactions ?? [];
      setRows(Array.isArray(txs) ? txs : []);
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
        <p className="text-slate-600">Sign in as a rider to view transactions.</p>
        <Button asChild>
          <Link href="/login?next=/finance/me">Sign in</Link>
        </Button>
      </div>
    );
  }

  if (session.user.role !== "customer") {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-slate-600">This page is for customer accounts.</p>
        <Button asChild variant="outline">
          <Link href="/">Home</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">My transactions</h1>
            <p className="text-sm text-slate-500">{session.user.email}</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" asChild>
              <Link href="/">Home</Link>
            </Button>
            <Button variant="ghost" onClick={() => { logout(); window.location.href = "/"; }}>
              Sign out
            </Button>
          </div>
        </div>

        {error && <p className="text-red-600 text-sm">{error}</p>}

        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-100 text-left text-slate-600">
              <tr>
                <th className="p-3">When</th>
                <th className="p-3">Type</th>
                <th className="p-3">Amount</th>
                <th className="p-3">Region</th>
                <th className="p-3">Trip</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr>
                  <td colSpan={5} className="p-8 text-center text-slate-500">
                    No transactions yet. Complete a paid ride to see ledger entries.
                  </td>
                </tr>
              )}
              {rows.map((t) => (
                <tr key={t.id || Math.random()} className="border-t border-slate-100">
                  <td className="p-3 whitespace-nowrap">{t.createdAtRfc3339 || "—"}</td>
                  <td className="p-3">{t.type || "—"}</td>
                  <td className="p-3">
                    {t.amountCents != null
                      ? `${(t.amountCents / 100).toFixed(2)} ${(t.currency || "usd").toUpperCase()}`
                      : "—"}
                  </td>
                  <td className="p-3">{t.region || "—"}</td>
                  <td className="p-3 font-mono text-xs">{t.sourceTripId || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
