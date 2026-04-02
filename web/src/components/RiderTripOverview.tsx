import { RouteFare, TripPreview, Driver } from "../types"
import { DriverList } from "./DriversList"
import { Card } from "./ui/card"
import { Button } from "./ui/button"
import { convertMetersToKilometers, convertSecondsToMinutes } from "../utils/math"
import { Skeleton } from "./ui/skeleton"
import { TripOverviewCard } from "./TripOverviewCard"
import { StripePaymentButton } from "./StripePaymentButton"
import { DriverCard } from "./DriverCard"
import { TripEvents, PaymentEventSessionCreatedData } from "../contracts"
import { useState, useEffect } from "react"

interface TripOverviewProps {
  trip: TripPreview | null;
  status: TripEvents | null;
  assignedDriver?: Driver | null;
  paymentSession?: PaymentEventSessionCreatedData | null;
  selectedFare?: RouteFare | null;
  onPackageSelect: (carPackage: RouteFare) => void;
  onCancel: () => void;
  onIncreaseFare?: (percentage: number) => void;
}

export const RiderTripOverview = ({
  trip,
  status,
  assignedDriver,
  paymentSession,
  selectedFare,
  onPackageSelect,
  onCancel,
  onIncreaseFare,
}: TripOverviewProps) => {
  const [timeLeft, setTimeLeft] = useState(120);
  const [timerEnded, setTimerEnded] = useState(false);

  useEffect(() => {
    if (status === TripEvents.Created) {
      setTimeLeft(120);
      setTimerEnded(false);
    }
  }, [status, trip?.tripID]);

  useEffect(() => {
    if (status === TripEvents.Created && timeLeft > 0) {
      const timer = setTimeout(() => setTimeLeft((prev) => prev - 1), 1000);
      return () => clearTimeout(timer);
    } else if (timeLeft === 0 && status === TripEvents.Created) {
      setTimerEnded(true);
    }
  }, [timeLeft, status]);

  const showIncreasePrompt = status === TripEvents.NoDriversFound || timerEnded;

  if (!trip) {
    return (
      <TripOverviewCard
        title="Start a trip"
        description="Click on the map to set a destination"
      />
    )
  }

  if (showIncreasePrompt && status !== TripEvents.Completed && status !== TripEvents.Cancelled && status !== TripEvents.DriverAssigned && status !== TripEvents.PaymentSessionCreated) {
    return (
      <TripOverviewCard
        title="No drivers available right now"
        description="Would you like to increase your fare to attract more drivers?"
      >
        <div className="flex flex-col gap-2 mt-4">
          <Button variant="default" onClick={() => { onIncreaseFare?.(10); setTimerEnded(false); }}>
            Increase Fare by 10%
          </Button>
          <Button variant="default" onClick={() => { onIncreaseFare?.(20); setTimerEnded(false); }}>
            Increase Fare by 20%
          </Button>
          <Button variant="outline" className="w-full" onClick={onCancel}>
            Cancel Request
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.PaymentSessionCreated && paymentSession) {
    return (
      <TripOverviewCard
        title="Payment Required"
        description="Please complete the payment to confirm your trip"
      >
        <div className="flex flex-col gap-4">
          <DriverCard driver={assignedDriver} />

          <div className="text-sm text-gray-500">
            <p>Amount: {paymentSession.amount} {paymentSession.currency}</p>
            <p>Trip ID: {paymentSession.tripID}</p>
          </div>
          <StripePaymentButton paymentSession={paymentSession} />
        </div>
      </TripOverviewCard>
    )
  }

  // Handled by showIncreasePrompt
  /*
  if (status === TripEvents.NoDriversFound) {
    return (
      <TripOverviewCard
        title="No drivers found"
        description="No drivers found for your trip, please try again later"
      >
        <Button variant="outline" className="w-full" onClick={onCancel}>
          Go back
        </Button>
      </TripOverviewCard>
    )
  }
  */

  if (status === TripEvents.DriverAssigned) {
    const otp: string | null = (trip as any)?.otp ?? null;
    return (
      <TripOverviewCard
        title="Driver assigned!"
        description="Your driver is on the way to pick you up."
      >
        <div className="flex flex-col gap-4">
          {/* Pulsing arrival badge */}
          <div className="flex flex-col items-center gap-3 py-2">
            <div className="relative flex items-center justify-center w-16 h-16">
              <span className="absolute inline-flex h-full w-full rounded-full bg-blue-300 opacity-40 animate-ping" />
              <span className="relative flex items-center justify-center w-12 h-12 rounded-full bg-blue-500 text-white text-2xl">🚗</span>
            </div>
            <div className="text-center">
              <p className="text-base font-semibold text-gray-800">Your driver is en route</p>
              <p className="text-sm text-gray-500 mt-1">Estimated arrival: <span className="font-bold text-blue-600">5–15 min</span></p>
            </div>
          </div>

          <div className="bg-blue-50 border border-blue-100 rounded-xl p-3 text-sm text-blue-700 text-center font-medium">
            📍 Driver is heading to your pickup location
          </div>

          {otp && (
            <div className="bg-amber-50 border-2 border-amber-300 rounded-xl p-4 text-center">
              <p className="text-xs text-amber-600 font-medium uppercase tracking-wide mb-1">Your ride OTP</p>
              <p className="text-3xl font-extrabold text-amber-700 tracking-widest">{otp}</p>
              <p className="text-xs text-amber-500 mt-1">Share this with your driver to confirm pickup</p>
            </div>
          )}
        </div>
        <div className="flex flex-col gap-2 mt-4">
          <Button variant="destructive" className="w-full" onClick={onCancel}>
            Cancel current trip
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Completed) {
    return (
      <TripOverviewCard
        title="Trip completed!"
        description="Your trip is completed, thank you for using our service!"
      >
        <Button variant="outline" className="w-full" onClick={onCancel}>
          Go back
        </Button>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Cancelled) {
    return (
      <TripOverviewCard
        title="Trip cancelled!"
        description="Your trip is cancelled, please try again later"
      >
        <Button variant="outline" className="w-full" onClick={onCancel}>
          Go back
        </Button>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Created || (trip.tripID && !status)) {
    return (
      <TripOverviewCard
        title="Looking for a driver"
        description={selectedFare ? `Finding the driver for the ${selectedFare.packageSlug}` : "Your trip is confirmed! We're matching you with a driver, it should not take long."}
      >
        <div className="flex flex-col space-y-3 justify-center items-center mb-4">
          <Skeleton className="h-[125px] w-[250px] rounded-xl" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-[250px]" />
            <Skeleton className="h-4 w-[200px]" />
          </div>
          {selectedFare && (
            <div className="text-xl font-bold text-gray-800 mt-4 bg-green-50 px-4 py-2 rounded-lg border border-green-200 shadow-sm transition-all duration-300 transform">
              Offering: ₹{(((selectedFare.totalPriceInCents || 0) * (selectedFare.packageSlug === 'carpool' ? (selectedFare.requestedSeats || 1) : 1)) / 100).toFixed(2)}
            </div>
          )}
          <div className="flex flex-col items-center justify-center mt-6 group relative">
            <div className="relative flex justify-center items-center">
              <svg width="64" height="64" className="transform -rotate-90 origin-center transition-all duration-500 ease-in-out">
                <circle cx="32" cy="32" r="28" className="text-gray-100" strokeWidth="6" stroke="currentColor" fill="transparent" />
                <circle cx="32" cy="32" r="28" className="text-blue-500 transition-all duration-1000 ease-linear shadow-blue-500/50" strokeWidth="6" stroke="currentColor" fill="transparent" strokeDasharray={2 * Math.PI * 28} strokeDashoffset={2 * Math.PI * 28 * ((120 - timeLeft) / 120)} strokeLinecap="round" />
              </svg>
              <div className="absolute flex flex-col items-center justify-center">
                <span className="text-lg font-extrabold text-gray-800">{timeLeft}s</span>
              </div>
            </div>
            <span className="text-xs text-gray-500 font-medium mt-2 tracking-wide uppercase">Searching</span>
          </div>
        </div>

        <div className="flex flex-col items-center justify-center gap-2">
          {trip?.duration &&
            <h3 className="text-sm font-medium text-gray-700 mb-2">Arriving in: {convertSecondsToMinutes(trip?.duration)} at your destination ({convertMetersToKilometers(trip?.distance ?? 0)})</h3>
          }

          <Button variant="destructive" className="w-full" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (trip.rideFares && trip.rideFares.length > 0 && !trip.tripID) {
    return (
      <DriverList
        trip={trip}
        onPackageSelect={onPackageSelect}
        onCancel={onCancel}
      />
    )
  }

  return (
    <Card className="w-full md:max-w-[500px] z-[9999] flex-[0.3] p-6 flex flex-col items-center justify-center text-center">
      <h3 className="text-lg font-semibold mb-2">No active trip found</h3>
      <p className="text-gray-500 mb-4">Please set a destination on the map to start.</p>
      <Button onClick={onCancel} variant="outline" className="w-full">Go Back</Button>
    </Card>
  )
}