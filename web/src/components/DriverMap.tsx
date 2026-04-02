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
import { TripEvents } from "../contracts";
import { useState, useMemo, useRef } from "react";
import { cn } from "../lib/utils";

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
    patchDriverSeats,
    acceptPendingRequest,
    declinePendingRequest,
    triedDriverIdsMap,
  } = useDriverStreamConnection({
    location: riderLocation,
    geohash: driverGeohash,
    userID,
    packageSlug,
  })

  const [isGPSTracking, setIsGPSTracking] = useState(false);

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

  const handleAcceptTrip = () => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set")
      return
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver: driver,
      }
    })

    // Optimistically decrement the driver's available seats
    if (requestedTrip.selectedFare?.packageSlug === CarPackageSlug.CARPOOL) {
      const seatsNeeded = requestedTrip.selectedFare?.requestedSeats ?? 1;
      patchDriverSeats(seatsNeeded);
    }

    setActiveTrip(requestedTrip);
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
              "flex items-center gap-3 px-6 py-3 rounded-full font-bold shadow-2xl transition-all duration-500 scale-100 active:scale-95 border-2",
              isGPSTracking
                ? "bg-green-600 text-white border-green-400 animate-pulse ring-4 ring-green-500/30"
                : "bg-white text-gray-700 border-gray-200 hover:border-blue-400"
            )}
          >
            <div className={cn(
              "w-3 h-3 rounded-full",
              isGPSTracking ? "bg-white animate-ping" : "bg-gray-300"
            )} />
            {isGPSTracking ? "GPS: LIVE TRACKING" : "GPS: START TRACKING"}
          </button>
          {!isGPSTracking && (
            <p className="text-[10px] bg-black/60 text-white px-3 py-1 rounded-full font-black tracking-widest backdrop-blur-sm">
              OR CLICK MAP TO SET MANUAL LOCATION
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
