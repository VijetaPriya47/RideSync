"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "../../../components/ui/button";
import { apiFetch } from "../../../lib/api";
import { useSession } from "../../../hooks/useSession";

type RideRow = {
  trip_id?: string;
  role?: string;
  status?: string;
  when_rfc3339?: string;
  fare_total_cents?: number;
  package_slug?: string;
  other_party_label?: string;
};

export default function FinanceMePage() {
  const { session, ready, logout } = useSession();
  const [rows, setRows] = useState<RideRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!ready || !session || session.user.role !== "customer") return;
    (async () => {
      const res = await apiFetch("/api/trips/history");
      const body = await res.json();
      if (!res.ok) {
        setError(body?.error?.message || "Failed to load");
        return;
      }
      const data = body.data;
      const entries = data?.entries ?? data?.Entries ?? [];
      setRows(Array.isArray(entries) ? entries : []);
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
        <p className="text-slate-600">Sign in as a rider to view your ride history.</p>
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
      <div className="max-w-5xl mx-auto space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">My ride history</h1>
            <p className="text-sm text-slate-500">{session.user.email}</p>
            <p className="text-xs text-slate-400 mt-1">
              Trips you booked as a rider and trips you accepted as a driver (any status).
            </p>
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
                <th className="p-3">Role</th>
                <th className="p-3">Status</th>
                <th className="p-3">Package</th>
                <th className="p-3">Fare</th>
                <th className="p-3">Details</th>
                <th className="p-3">Trip ID</th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr>
                  <td colSpan={7} className="p-8 text-center text-slate-500">
                    No rides yet. Book a trip as a rider or accept one as a driver to see it here.
                  </td>
                </tr>
              )}
              {rows.map((t) => (
                <tr key={t.trip_id || Math.random()} className="border-t border-slate-100">
                  <td className="p-3 whitespace-nowrap">{t.when_rfc3339 || "—"}</td>
                  <td className="p-3 capitalize">{t.role || "—"}</td>
                  <td className="p-3 capitalize">{t.status || "—"}</td>
                  <td className="p-3">{t.package_slug || "—"}</td>
                  <td className="p-3">
                    {t.fare_total_cents != null
                      ? `${(t.fare_total_cents / 100).toFixed(2)}`
                      : "—"}
                  </td>
                  <td className="p-3 text-slate-600 max-w-[200px] truncate" title={t.other_party_label}>
                    {t.other_party_label || "—"}
                  </td>
                  <td className="p-3 font-mono text-xs">{t.trip_id || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
