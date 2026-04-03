import React from "react"
import { Trip, Route } from "../types"
import { TripOverviewCard } from "./TripOverviewCard"
import { Button } from "./ui/button"
import { TripEvents } from "../contracts"
import { haversineDistanceKm } from "../utils/math"

// Bounding box overlap heuristic: check if any point in the new request's route
// falls within the accepted trip's route bounding box (± 0.005 deg ≈ 0.5 km)
function routesOverlap(acceptedRoute?: Route, newRoute?: Route): boolean {
  if (!acceptedRoute || !newRoute) return true; // optimistic if data missing
  const allCoords = acceptedRoute.geometry.flatMap((g) => g.coordinates);
  if (allCoords.length === 0) return true;
  const lats = allCoords.map((c) => c.latitude);
  const lons = allCoords.map((c) => c.longitude);
  const TOLERANCE = 0.005;
  const minLat = Math.min(...lats) - TOLERANCE;
  const maxLat = Math.max(...lats) + TOLERANCE;
  const minLon = Math.min(...lons) - TOLERANCE;
  const maxLon = Math.max(...lons) + TOLERANCE;

  return newRoute.geometry.some((g) =>
    g.coordinates.some(
      (c) => c.latitude >= minLat && c.latitude <= maxLat && c.longitude >= minLon && c.longitude <= maxLon
    )
  );
}

interface DriverTripOverviewProps {
  trip?: Trip | null;
  status?: TripEvents | null;
  onAcceptTrip?: () => void;
  onDeclineTrip?: () => void;
  pendingCarpoolRequests?: Trip[];
  availableSeats?: number;
  activeTrip?: Trip | null;
  onAcceptPending?: (trip: Trip) => void;
  onDeclinePending?: (trip: Trip) => void;
  /** Driver's current GPS location */
  driverLocation?: { latitude: number; longitude: number } | null;
  /** Pickup location from the requested trip's route */
  pickupLocation?: { latitude: number; longitude: number } | null;
}

export const DriverTripOverview = ({
  trip,
  status,
  onAcceptTrip,
  onDeclineTrip,
  pendingCarpoolRequests = [],
  availableSeats,
  activeTrip,
  onAcceptPending,
  onDeclinePending,
  driverLocation,
  pickupLocation,
}: DriverTripOverviewProps) => {
  const [timeLeft, setTimeLeft] = React.useState(120);

  React.useEffect(() => {
    if (status === TripEvents.DriverTripRequest && trip) {
      setTimeLeft(120);
    }
  }, [status, trip?.id]);

  React.useEffect(() => {
    if (status === TripEvents.DriverTripRequest && timeLeft > 0) {
      const timer = setTimeout(() => setTimeLeft((prev) => prev - 1), 1000);
      return () => clearTimeout(timer);
    } else if (timeLeft === 0 && status === TripEvents.DriverTripRequest) {
      onDeclineTrip?.();
    }
  }, [timeLeft, status]);

  // If we have an active trip, show the dashboard (All set!)
  // If we don't have an active trip but have a request, show the request overlay
  if (!activeTrip && !trip) {
    return (
      <TripOverviewCard
        title="Waiting for a rider..."
        description="Waiting for a rider to request a trip..."
      />
    )
  }

  if (status === TripEvents.DriverTripRequest && trip && !activeTrip) {
    const seatsRequested = trip.selectedFare?.requestedSeats || 1;
    const rawFare = trip.selectedFare?.totalPriceInCents || 0;
    const formattedFare = ((rawFare * seatsRequested) / 100).toFixed(2);

    // Distance from driver to pickup
    const distanceToPickup =
      driverLocation && pickupLocation
        ? haversineDistanceKm(driverLocation.latitude, driverLocation.longitude, pickupLocation.latitude, pickupLocation.longitude)
        : null;
    const etaMinutes = distanceToPickup !== null ? Math.max(1, Math.round((distanceToPickup / 30) * 60)) : null;

    return (
      <TripOverviewCard
        title="Trip request received!"
        description="A trip has been requested, check the route and accept the trip if you can take it."
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-col items-center justify-center -mt-2 mb-2">
            <div className="text-2xl font-black text-green-700 bg-green-50 px-6 py-3 rounded-2xl border-2 border-green-200 shadow-sm text-center transition-all duration-300 transform font-mono">
              Offered Fare: ₹{rawFare > 0 ? formattedFare : "Calculating..."}
            </div>
            {trip.selectedFare?.packageSlug === 'carpool' && (
              <p className="text-sm font-bold text-blue-600 mt-2 flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
                Carpool: {seatsRequested} seat{seatsRequested !== 1 ? 's' : ''}
              </p>
            )}
            {distanceToPickup !== null && (
              <div className="flex items-center gap-2 mt-2 bg-blue-50 border border-blue-200 px-4 py-2 rounded-xl text-sm font-semibold text-blue-700">
                <span>📍</span>
                <span>{distanceToPickup.toFixed(1)} km away · ~{etaMinutes} min to pickup</span>
              </div>
            )}
          </div>

          <div className="flex flex-col items-center justify-center group relative py-2">
            <div className="relative flex justify-center items-center">
              <svg width="72" height="72" className="transform -rotate-90 origin-center transition-all duration-500 ease-in-out">
                <circle cx="36" cy="36" r="32" className="text-gray-100" strokeWidth="8" stroke="currentColor" fill="transparent" />
                <circle cx="36" cy="36" r="32" className="text-blue-500 transition-all duration-1000 ease-linear" strokeWidth="8" stroke="currentColor" fill="transparent" strokeDasharray={2 * Math.PI * 32} strokeDashoffset={2 * Math.PI * 32 * ((120 - timeLeft) / 120)} strokeLinecap="round" />
              </svg>
              <div className="absolute flex flex-col items-center justify-center">
                <span className="text-xl font-black text-gray-800">{timeLeft}s</span>
              </div>
            </div>
            <span className="text-[10px] text-gray-400 font-bold mt-2 tracking-widest uppercase">Expires Promptly</span>
          </div>

          <div className="flex flex-col gap-2 w-full mt-2">
            <Button size="lg" className="h-12 text-lg font-bold shadow-lg shadow-blue-200" onClick={onAcceptTrip}>
              Accept Trip
            </Button>
            <Button size="lg" variant="outline" className="h-12 text-lg font-bold border-2" onClick={onDeclineTrip}>
              Pass
            </Button>
          </div>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.DriverTripAccept || activeTrip) {
    const displayTrip = activeTrip || trip;
    if (!displayTrip) return null;
    const overlappingRequests = pendingCarpoolRequests.filter((req) =>
      routesOverlap(displayTrip.route, req.route)
    );
    const hasSeats = availableSeats !== undefined && availableSeats > 0;

    return (
      <div className="flex flex-col gap-3">
        <TripOverviewCard
          title="All set!"
          description="You can now start the trip"
        >
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <h3 className="text-lg font-bold">Trip details</h3>
              <p className="text-sm text-gray-500">
                Trip ID: {displayTrip.id}
                <br />
                Rider ID: {displayTrip.userID}
                {displayTrip.selectedFare?.requestedSeats && (
                  <>
                    <br />
                    Requested Seats: {displayTrip.selectedFare.requestedSeats}
                  </>
                )}
              </p>
            </div>
            {hasSeats && (
              <div className="text-center text-xs font-semibold text-green-700 bg-green-50 border border-green-200 px-3 py-1.5 rounded-full">
                🟢 {availableSeats} seat{availableSeats !== 1 ? 's' : ''} available for carpool
              </div>
            )}
          </div>
        </TripOverviewCard>

        {hasSeats && overlappingRequests.length > 0 && (
          <div className="bg-white rounded-2xl shadow-md border border-blue-100 p-4">
            <h3 className="text-base font-semibold text-gray-800 mb-3 flex items-center gap-2">
              <span className="inline-block w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
              Incoming Carpool Requests
            </h3>
            <div className="flex flex-col gap-3">
              {overlappingRequests.map((req) => {
                const seatsNeeded = req.selectedFare?.requestedSeats || 1;
                const fare = req.selectedFare?.totalPriceInCents
                  ? ((req.selectedFare.totalPriceInCents * seatsNeeded) / 100).toFixed(2)
                  : "—";
                const canFit = seatsNeeded <= (availableSeats ?? 0);
                return (
                  <div
                    key={req.id}
                    className="border border-gray-100 rounded-xl p-3 bg-gray-50 flex flex-col gap-2"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-gray-800">
                          {seatsNeeded} seat{seatsNeeded !== 1 ? 's' : ''} requested
                        </p>
                        <p className="text-xs text-gray-400 truncate max-w-[180px]">Rider: {req.userID}</p>
                      </div>
                      <div className="flex flex-col items-end gap-1">
                        <span className="text-base font-bold text-green-700 font-mono">₹{fare}</span>
                        <span className="text-xs bg-green-50 text-green-700 border border-green-200 rounded-full px-2 py-0.5 font-medium">
                          Route match ✓
                        </span>
                      </div>
                    </div>
                    {!canFit && (
                      <p className="text-xs text-amber-600 font-medium">
                        ⚠ Needs {seatsNeeded} seats, only {availableSeats} available
                      </p>
                    )}
                    <div className="flex gap-2">
                      <Button
                        className="flex-1 text-sm h-8"
                        disabled={!canFit}
                        onClick={() => onAcceptPending?.(req)}
                      >
                        Accept
                      </Button>
                      <Button
                        variant="outline"
                        className="flex-1 text-sm h-8"
                        onClick={() => onDeclinePending?.(req)}
                      >
                        Decline
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {hasSeats && overlappingRequests.length === 0 && pendingCarpoolRequests.length > 0 && (
          <div className="bg-amber-50 border border-amber-200 rounded-xl p-3 text-sm text-amber-700 font-medium text-center">
            {pendingCarpoolRequests.length} request(s) pending — routes do not overlap
          </div>
        )}
      </div>
    )
  }

  return null
}
