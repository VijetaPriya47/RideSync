"use client"

import { useDriverStreamConnection } from "../hooks/useDriverStreamConnection"
import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet'
import L from 'leaflet';
import { MapClickHandler } from './MapClickHandler';
import { CarPackageSlug, Coordinate } from "../types";
import { DriverTripOverview } from "./DriverTripOverview";
import * as Geohash from 'ngeohash';
import { RoutingControl } from "./RoutingControl";
import { DriverCard } from "./DriverCard";
import { TripEvents, BackendEndpoints } from "../contracts";
import { useState, useMemo, useRef, useEffect } from "react";
import { cn } from "../lib/utils";
import { API_URL } from "../constants";
import { LocateFixed } from "lucide-react";

const START_LOCATION: Coordinate = {
  latitude: 28.6139,
  longitude: 77.2090,
}

const driverMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
  iconSize: [30, 30],
  iconAnchor: [15, 30],
});

const startLocationMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/535711/user.svg",
  iconSize: [30, 40], // Size of the marker
  iconAnchor: [20, 40], // Anchor point
});

const destinationMarker = new L.Icon({
  iconUrl: "https://upload.wikimedia.org/wikipedia/commons/thumb/e/ed/Map_pin_icon.svg/176px-Map_pin_icon.svg.png",
  iconSize: [40, 40], // Size of the marker
  iconAnchor: [20, 40], // Anchor point
});

export const DriverMap = ({ packageSlug }: { packageSlug: CarPackageSlug }) => {
  const mapRef = useRef<L.Map>(null)
  const userID = useMemo(() => crypto.randomUUID(), [])
  const [riderLocation, setRiderLocation] = useState<Coordinate>(START_LOCATION)
  const [completedTrip, setCompletedTrip] = useState<{
    tripId: string;
    riderId: string;
    amount: string;
  } | null>(null)

  const driverGeohash = useMemo(() =>
    Geohash.encode(riderLocation?.latitude, riderLocation?.longitude, 7)
    , [riderLocation?.latitude, riderLocation?.longitude]);

  const {
    error,
    driver,
    tripStatus,
    requestedTrip,
    activeTrip,
    setActiveTrip,
    pendingCarpoolRequests,
    sendMessage,
    setTripStatus,
    resetTripStatus,
    reserveSeatsForAcceptedTrip,
    restoreSeatsAfterTrip,
    acceptPendingRequest,
    declinePendingRequest,
    triedDriverIdsMap,
  } = useDriverStreamConnection({
    location: riderLocation,
    geohash: driverGeohash,
    userID,
    packageSlug,
  })

  const [isGPSTracking, setIsGPSTracking] = useState(true);

  // GPS Tracking logic
  useMemo(() => {
    if (!isGPSTracking) return;

    const watchId = navigator.geolocation.watchPosition(
      (position) => {
        setRiderLocation({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        });
      },
      (error) => {
        console.error("GPS Tracking Error:", error);
        setIsGPSTracking(false);
        alert("Could not access GPS. Please check permissions.");
      },
      { enableHighAccuracy: true }
    );

    return () => navigator.geolocation.clearWatch(watchId);
  }, [isGPSTracking]);

  const handleMapClick = (e: L.LeafletMouseEvent) => {
    if (isGPSTracking) return; // Disable manual clicks when GPS is on
    setRiderLocation({
      latitude: e.latlng.lat,
      longitude: e.latlng.lng
    })
  }

  const handleAcceptTrip = async () => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set")
      return
    }

    try {
      const url = `${API_URL}${BackendEndpoints.GET_TRIP}`.replace('{id}', requestedTrip.id);
      const statusResp = await fetch(url);
      if (statusResp.ok) {
        const { data } = await statusResp.json();
        if (data.status === 'accepted' || data.status === 'completed' || data.status === 'cancelled') {
          alert(`Cannot accept trip. Trip is already ${data.status}.`);
          return;
        }
      }
    } catch (e) {
      console.error("Failed to check trip status", e);
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver: driver,
      }
    })

    reserveSeatsForAcceptedTrip(requestedTrip);

    setActiveTrip(requestedTrip);
    setCompletedTrip(null);
    // Explicitly clear requested trip after setting active trip
    // so we don't have multiple trips in the overlay
    setTripStatus(TripEvents.DriverTripAccept);
  };

  const handleDeclineTrip = () => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set")
      return
    }

    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver: driver,
        triedDriverIds: triedDriverIdsMap[requestedTrip.id] || [],
      }
    })

    setTripStatus(TripEvents.DriverTripDecline)
    resetTripStatus()
  }

  console.log({ requestedTrip })

  // destination is the last coordinate in the route
  const destination = useMemo(() => {
    const geoLen = requestedTrip?.route?.geometry?.length || 0;
    if (geoLen === 0) return null;
    const coords = requestedTrip?.route?.geometry[geoLen - 1]?.coordinates;
    if (!coords || coords.length === 0) return null;
    return coords[coords.length - 1];
  }, [requestedTrip])

  // start location is the first coordinate in the route
  const startLocation = useMemo(() => {
    const coords = requestedTrip?.route?.geometry?.[0]?.coordinates;
    if (!coords || coords.length === 0) return null;
    return coords[0];
  }, [requestedTrip])

  // Get active trip details for carpool rendering
  const activeTripIds = useMemo(() => driver?.activeTripIds || [], [driver]);
  const isCarpool = packageSlug === CarPackageSlug.CARPOOL;

  useEffect(() => {
    if (activeTrip || activeTripIds.length === 0) return;

    let cancelled = false;

    const hydrateActiveTrip = async () => {
      try {
        const url = `${API_URL}${BackendEndpoints.GET_TRIP}`.replace('{id}', activeTripIds[0]);
        const response = await fetch(url);
        if (!response.ok) return;

        const { data } = await response.json();
        if (!cancelled && data) {
          setActiveTrip(data);
          setTripStatus(TripEvents.DriverTripAccept);
        }
      } catch (error) {
        console.error("Failed to hydrate active trip", error);
      }
    };

    hydrateActiveTrip();

    return () => {
      cancelled = true;
    };
  }, [activeTrip, activeTripIds, setActiveTrip, setTripStatus]);

  useEffect(() => {
    if (!activeTrip?.id) return;

    let cancelled = false;

    const syncActiveTripStatus = async () => {
      try {
        const url = `${API_URL}${BackendEndpoints.GET_TRIP}`.replace('{id}', activeTrip.id);
        const response = await fetch(url);
        if (!response.ok) return;

        const { data } = await response.json();
        if (cancelled || !data?.status) return;

        if (data.status === 'payed' || data.status === 'completed') {
          const seatsMultiplier = activeTrip.selectedFare?.requestedSeats ?? 1;
          const totalAmount = (((activeTrip.selectedFare?.totalPriceInCents ?? 0) * seatsMultiplier) / 100).toFixed(2);
          restoreSeatsAfterTrip(activeTrip);
          setCompletedTrip({
            tripId: activeTrip.id,
            riderId: activeTrip.userID,
            amount: totalAmount,
          });
          setActiveTrip(null);
          setTripStatus(TripEvents.Completed);
          return;
        }

        if (data.status === 'cancelled') {
          restoreSeatsAfterTrip(activeTrip);
          setActiveTrip(null);
          setTripStatus(TripEvents.Cancelled);
        }
      } catch (error) {
        console.error("Failed to sync active trip status", error);
      }
    };

    const intervalId = window.setInterval(syncActiveTripStatus, 4000);
    syncActiveTripStatus();

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, [activeTrip, restoreSeatsAfterTrip, setActiveTrip, setTripStatus]);


  if (error) {
    return <div>Error: {error}</div>
  }

  return (
    <div className="relative flex flex-col md:flex-row h-screen">
      <div className="flex-1">
        <MapContainer
          center={[riderLocation.latitude, riderLocation.longitude]}
          zoom={13}
          style={{ height: '100%', width: '100%' }}
          ref={mapRef}
        >
          <TileLayer
            url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
            attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
          />

          <Marker
            key={userID}
            position={[riderLocation.latitude, riderLocation.longitude]}
            icon={driverMarker}
          >
            <Popup>
              Driver ID: {userID}
              <br />
              Geohash: {driverGeohash}
            </Popup>
          </Marker>

          {startLocation && (
            <Marker position={[startLocation.longitude, startLocation.latitude]} icon={startLocationMarker}>
              <Popup>Start Location</Popup>
            </Marker>
          )}

          {destination && (
            <Marker position={[destination.longitude, destination.latitude]} icon={destinationMarker}>
              <Popup>Destination</Popup>
            </Marker>
          )}

          {isCarpool && activeTripIds.length > 0 && (
            <Marker position={[riderLocation.latitude, riderLocation.longitude]} icon={driverMarker}>
              <Popup>
                Active Carpool Trips: {activeTripIds.length} <br />
                Available Seats: {driver?.availableSeats}
              </Popup>
            </Marker>
          )}

          {requestedTrip?.route && (
            <RoutingControl route={requestedTrip.route} />
          )}

          <MapClickHandler onClick={handleMapClick} />
        </MapContainer>

        {/* GPS Tracking Floating Toggle */}
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-[1000] flex flex-col items-center gap-2">
          <button
            onClick={() => setIsGPSTracking(!isGPSTracking)}
            className={cn(
              "flex items-center gap-3 rounded-full border bg-white/95 px-4 py-2.5 text-sm font-semibold text-gray-800 shadow-lg backdrop-blur-sm transition-all duration-300 active:scale-95",
              isGPSTracking
                ? "border-blue-200 ring-1 ring-blue-100"
                : "border-gray-200 hover:border-blue-300 hover:shadow-xl"
            )}
          >
            <span
              className={cn(
                "flex h-9 w-9 items-center justify-center rounded-full border",
                isGPSTracking
                  ? "border-blue-200 bg-blue-50 text-blue-600"
                  : "border-gray-200 bg-gray-50 text-gray-500"
              )}
            >
              <LocateFixed className={cn("h-4 w-4", isGPSTracking && "animate-pulse")} />
            </span>
            <span className="flex flex-col items-start leading-tight">
              <span>{isGPSTracking ? "GPS Tracking" : "Location Paused"}</span>
              <span className="text-xs font-medium text-gray-500">
                {isGPSTracking ? "Live location on" : "Tap to resume live GPS"}
              </span>
            </span>
          </button>
          {!isGPSTracking && (
            <p className="rounded-full bg-white/95 px-3 py-1 text-[10px] font-semibold tracking-wide text-gray-600 shadow-md backdrop-blur-sm">
              Click the map to place your driver manually
            </p>
          )}
        </div>
      </div>

      <div className="flex flex-col w-full md:w-[450px] bg-white border-t md:border-t-0 md:border-l border-gray-100">
        <div className="p-4 border-b">
          <DriverCard driver={driver} packageSlug={packageSlug} />
        </div>
        <div className="flex-1 overflow-y-auto">
          <DriverTripOverview
            trip={requestedTrip}
            status={tripStatus}
            onAcceptTrip={handleAcceptTrip}
            onDeclineTrip={handleDeclineTrip}
            pendingCarpoolRequests={pendingCarpoolRequests}
            availableSeats={driver?.availableSeats}
            activeTrip={activeTrip}
            completedTrip={completedTrip}
            onAcceptPending={acceptPendingRequest}
            onDeclinePending={declinePendingRequest}
            driverLocation={riderLocation}
            pickupLocation={
              requestedTrip?.route?.geometry?.[0]?.coordinates?.[0]
                ? { latitude: requestedTrip.route.geometry[0].coordinates[0].latitude, longitude: requestedTrip.route.geometry[0].coordinates[0].longitude }
                : null
            }
          />
        </div>
      </div>
    </div>
  )
}
