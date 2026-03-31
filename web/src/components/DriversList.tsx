import { Button } from "./ui/button"
import { Clock, Minus, Plus, Users } from 'lucide-react'
import { RouteFare, TripPreview } from '../types'
import { convertMetersToKilometers, convertSecondsToMinutes } from "../utils/math"
import { cn } from "../lib/utils"
import { PackagesMeta } from "./PackagesMeta"
import { useState } from "react"
import { API_URL } from "../constants"
import { BackendEndpoints } from "../contracts"

interface DriverListProps {
  trip: TripPreview | null;
  onPackageSelect: (fare: RouteFare) => void
  onCancel: () => void
}

export function DriverList({ trip, onPackageSelect, onCancel }: DriverListProps) {
  const [carpoolSeats, setCarpoolSeats] = useState(1);
  const [isUpdating, setIsUpdating] = useState(false);

  const handleSelect = async (fare: RouteFare) => {
    if (fare.packageSlug === 'carpool') {
      setIsUpdating(true);
      try {
        await fetch(`${API_URL}${BackendEndpoints.UPDATE_TRIP_SEATS}`, {
          method: 'POST',
          body: JSON.stringify({ fareID: fare.id, seats: carpoolSeats }),
          headers: { 'Content-Type': 'application/json' }
        });
        fare.requestedSeats = carpoolSeats;
      } catch (e) {
        console.error("Failed to update seats", e);
      } finally {
        setIsUpdating(false);
      }
    }
    onPackageSelect(fare);
  }

  return (
    <div className="flex items-center justify-center p-4 min-h-screen bg-black/20">
      <div className="bg-white rounded-2xl shadow-lg p-6 max-w-md w-full">
        <h2 className="text-xl font-semibold mb-2">Select your desired ride</h2>
        <p className="text-sm text-gray-500 mb-6">Routing for {convertMetersToKilometers(trip?.distance ?? 0)}</p>
        <div className="flex items-center gap-1 text-sm text-gray-500 mb-2">
          <Clock className="w-4 h-4" />
          <span>You&apos;ll arrive in: {convertSecondsToMinutes(trip?.duration ?? 0)}</span>
        </div>
        <div className="space-y-4">
          {trip?.rideFares.map((fare) => {
            const Icon = PackagesMeta[fare.packageSlug].icon;
            const isCarpool = fare.packageSlug === 'carpool';
            const multiplier = isCarpool ? carpoolSeats : 1;
            const price = fare.totalPriceInCents && `$${((fare.totalPriceInCents * multiplier) / 100).toFixed(2)}`

            return (
              <div
                key={fare.id}
                className={cn(
                  "flex flex-col p-4 rounded-lg border transition-all",
                  "hover:border-primary hover:bg-primary/5 group relative",
                )}
              >
                <div className="flex items-center justify-between w-full mb-3">
                  <div className="flex items-center gap-4">
                    <div className="p-2 bg-gray-100 rounded-lg">
                      {Icon}
                    </div>
                    <div>
                      <h3 className="font-medium">{PackagesMeta[fare.packageSlug].name}</h3>
                      <p className="text-sm text-gray-500">{PackagesMeta[fare.packageSlug].description}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold">{price}</p>
                  </div>
                </div>

                {isCarpool && (
                  <div className="flex items-center justify-between bg-gray-50 p-2 rounded-md mb-3">
                    <div className="flex items-center gap-2 text-sm text-gray-600 font-medium">
                      <Users className="w-4 h-4 text-primary" />
                      How many seats?
                    </div>
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => setCarpoolSeats(Math.max(1, carpoolSeats - 1))}
                        className="w-8 h-8 rounded-full border border-gray-200 flex items-center justify-center hover:bg-white active:scale-95 transition-all text-gray-600"
                      >
                        <Minus className="w-3 h-3" />
                      </button>
                      <span className="w-4 text-center font-bold text-primary">{carpoolSeats}</span>
                      <button
                        onClick={() => setCarpoolSeats(Math.min(4, carpoolSeats + 1))}
                        className="w-8 h-8 rounded-full border border-gray-200 flex items-center justify-center hover:bg-white active:scale-95 transition-all text-gray-600"
                      >
                        <Plus className="w-3 h-3" />
                      </button>
                    </div>
                  </div>
                )}

                <Button
                  onClick={() => handleSelect(fare)}
                  disabled={isUpdating}
                  className="w-full mt-1"
                >
                  {isUpdating ? "Syncing..." : isCarpool ? `Book Carpool (${carpoolSeats} Seats)` : `Select ${PackagesMeta[fare.packageSlug].name}`}
                </Button>
              </div>
            );
          })}
        </div>
        <div className="mt-6">
          <Button
            variant="outline"
            className="w-full"
            onClick={() => onCancel()}
          >
            Back to Map
          </Button>
        </div>
      </div>
    </div>
  )
}
