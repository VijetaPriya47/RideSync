"use client"

// Assets
import 'leaflet/dist/leaflet.css';
// Fix for default marker icon
import icon from 'leaflet/dist/images/marker-icon.png'
import iconShadow from 'leaflet/dist/images/marker-shadow.png'
import dynamic from 'next/dynamic'
import Link from "next/link";
import { Button } from "../components/ui/button";
import { useState, Suspense, type ReactNode } from "react";
import { useSearchParams, useRouter } from 'next/navigation';
import { CarPackageSlug } from '../types';
import { DriverPackageSelector } from '../components/DriverPackageSelector';
import { useSession } from "../hooks/useSession";

// Dynamically import components that use Leaflet
const DriverMap = dynamic(() => import("../components/DriverMap").then(mod => mod.DriverMap), { ssr: false })
const RiderMap = dynamic(() => import("../components/RiderMap"), { ssr: false })

// Initialize Leaflet icon only on client side
if (typeof window !== 'undefined') {
  import('leaflet').then((L) => {
    const DefaultIcon = L.default.icon({
      iconUrl: icon.src,
      shadowUrl: iconShadow.src,
      iconSize: [25, 41],
      iconAnchor: [12, 41],
    })
    L.default.Marker.prototype.options.icon = DefaultIcon
  })
}

function HomeContent() {
  const [userType, setUserType] = useState<"driver" | "rider" | null>(null)
  const router = useRouter()
  const searchParams = useSearchParams()
  const payment = searchParams.get("payment")
  const [packageSlug, setPackageSlug] = useState<CarPackageSlug | null>(null)
  const { session, ready, logout } = useSession()

  const handleClick = (userType: "driver" | "rider") => {
    setUserType(userType)
  }

  const renderCustomerOnly = (render: (userId: string) => ReactNode) => {
    if (!ready) {
      return (
        <div className="flex justify-center items-center min-h-[40vh] text-gray-500">Loading…</div>
      )
    }
    if (!session || session.user.role !== "customer") {
      return (
        <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4 px-4">
          <p className="text-gray-600 text-center max-w-md">
            Trip booking and driving require a rider account. Sign in with Google (or your customer account).
          </p>
          <Button asChild>
            <Link href="/login?next=/">Sign in</Link>
          </Button>
          <Button variant="ghost" onClick={() => { setUserType(null); setPackageSlug(null); }}>
            Back
          </Button>
        </div>
      )
    }
    return render(session.user.id)
  }

  if (payment === 'success') {
    return (
      <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
        <div className="flex flex-col items-center justify-center h-screen gap-6 px-4">
          <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
            <div className="mb-6">
              <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-gray-900">Payment Successful!</h1>
              <p className="text-gray-600 mt-2">Your ride has been confirmed.</p>
            </div>
            <Button
              className="w-full text-lg py-6"
              variant="outline"
              onClick={() => router.push("/")}
            >
              Return Home
            </Button>
          </div>
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
      {userType === null && (
        <div className="flex flex-col items-center justify-center h-screen gap-6 px-4">
          <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
            <h2 className="text-2xl font-bold text-gray-900 mb-6">Welcome to RideSync</h2>
            <p className="text-gray-600 mb-4">Choose how you&apos;d like to use our service today</p>
            <div className="flex flex-wrap items-center justify-center gap-2 text-sm text-gray-600 mb-6 border-b border-gray-100 pb-4">
              {session ? (
                <>
                  <span className="text-xs text-gray-500 truncate max-w-[200px]">{session.user.email}</span>
                  <span className="text-xs rounded-full bg-gray-100 px-2 py-0.5">{session.user.role}</span>
                  {session.user.role === "customer" && (
                    <Link href="/finance/me" className="text-primary hover:underline">Ride history</Link>
                  )}
                  {(session.user.role === "business" || session.user.role === "admin") && (
                    <Link href="/dashboard" className="text-primary hover:underline">Dashboard</Link>
                  )}
                  {session.user.role === "admin" && (
                    <Link href="/admin" className="text-primary hover:underline">Admin</Link>
                  )}
                  <button type="button" className="text-red-600 hover:underline ml-1" onClick={() => logout()}>
                    Sign out
                  </button>
                </>
              ) : (
                <Link href="/login" className="text-primary font-medium hover:underline">Sign in</Link>
              )}
            </div>
            <div className="space-y-4">
              <Button
                className="w-full text-lg py-6 bg-primary hover:bg-primary/90"
                onClick={() => handleClick("rider")}
              >
                I Need a Ride
              </Button>
              <Button
                className="w-full text-lg py-6"
                variant="outline"
                onClick={() => handleClick("driver")}
              >
                I Want to Drive
              </Button>
              <Button
                className="w-full text-lg py-6 border-2 border-primary/20 text-primary hover:bg-primary/10"
                variant="ghost"
                onClick={() => window.open("https://vijetapriya47.github.io/RideSync/", "_blank")}
              >
                Documentation Website
              </Button>
            </div>
          </div>
        </div>
      )}

      {userType === "driver" && packageSlug && renderCustomerOnly((uid) => (
        <DriverMap packageSlug={packageSlug} userId={uid} />
      ))}

      {userType === "driver" && !packageSlug && renderCustomerOnly(() => (
        <DriverPackageSelector onSelect={setPackageSlug} />
      ))}

      {userType === "rider" && renderCustomerOnly((uid) => (
        <RiderMap userId={uid} />
      ))}
    </main>
  );
}

export default function Home() {
  return (
    <Suspense fallback={
      <main className="min-h-screen bg-gradient-to-b from-white to-gray-50">
        <div className="flex flex-col items-center justify-center h-screen gap-4">
          <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
            <div className="animate-pulse flex flex-col items-center">
              <div className="h-8 w-32 bg-gray-200 rounded mb-4"></div>
              <div className="h-4 w-48 bg-gray-100 rounded"></div>
            </div>
          </div>
        </div>
      </main>
    }>
      <HomeContent />
    </Suspense>
  );
}
