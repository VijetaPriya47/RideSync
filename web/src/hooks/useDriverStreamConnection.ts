import React, { useEffect, useState } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { Trip, Driver, CarPackageSlug } from '../types';
import { ServerWsMessage, TripEvents, isValidWsMessage, isValidTripEvent, ClientWsMessage, BackendEndpoints } from '../contracts';

interface useDriverConnectionProps {
  location: {
    latitude: number;
    longitude: number;
  };
  geohash: string;
  userID: string;
  packageSlug: CarPackageSlug;
}

export const useDriverStreamConnection = ({
  location,
  geohash,
  userID,
  packageSlug
}: useDriverConnectionProps) => {
  const [requestedTrip, setRequestedTrip] = useState<Trip | null>(null)
  const [activeTrip, setActiveTrip] = useState<Trip | null>(null)
  const [pendingCarpoolRequests, setPendingCarpoolRequests] = useState<Trip[]>([]);
  const [triedDriverIdsMap, setTriedDriverIdsMap] = useState<Record<string, string[]>>({});
  const [tripStatus, setTripStatus] = useState<TripEvents | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [driver, setDriver] = useState<Driver | null>(null);
  // Ref to always read the latest activeTrip status inside the WS callback
  const activeTripRef = React.useRef<Trip | null>(null);
  React.useEffect(() => { activeTripRef.current = activeTrip; }, [activeTrip]);
  const driverRef = React.useRef<Driver | null>(null);
  React.useEffect(() => { driverRef.current = driver; }, [driver]);

  useEffect(() => {
    if (ws?.readyState === WebSocket.OPEN && location && geohash) {
      ws.send(JSON.stringify({
        type: TripEvents.DriverLocation,
        data: {
          location,
          geohash,
        }
      }));
    }
  }, [location, geohash, ws?.readyState]);

  useEffect(() => {
    if (!userID) return;

    const websocket = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?userID=${userID}&packageSlug=${packageSlug}`);
    setWs(websocket);

    websocket.onopen = () => {
      if (location) {
        // Send initial location
        websocket.send(JSON.stringify({
          type: TripEvents.DriverLocation,
          data: {
            location,
            geohash,
          }
        }));
      }
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        setError(`Unknown message type "${message}", allowed types are: ${Object.values(TripEvents).join(', ')}`);
        return;
      }

      switch (message.type) {
        case TripEvents.DriverTripRequest: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const payload = message.data as any;
          const trip = payload.trip ?? payload;
          if (payload.triedDriverIds) {
            setTriedDriverIdsMap((prev: Record<string, string[]>) => ({ ...prev, [trip.id]: payload.triedDriverIds }));
          }
          if (activeTripRef.current || (driverRef.current?.activeTripIds?.length ?? 0) > 0) {
            setPendingCarpoolRequests((prev: Trip[]) =>
              prev.some((existing) => existing.id === trip.id) ? prev : [...prev, trip]
            );
          } else {
            setRequestedTrip(trip);
          }
          break;
        }
        case TripEvents.DriverRegister:
          setDriver(message.data);
          break;
      }


      if (isValidTripEvent(message.type)) {
        setTripStatus(message.type);
      } else {
        setError(`Unknown message type "${message.type}", allowed types are: ${Object.values(TripEvents).join(', ')}`);
      }
    };

    websocket.onclose = () => {
      console.log('WebSocket closed');
    };

    websocket.onerror = (event) => {
      setError('WebSocket error occurred');
      console.error('WebSocket error:', event);
    };

    return () => {
      console.log('Closing WebSocket');
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID]);

  const sendMessage = (message: ClientWsMessage) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      setError('WebSocket is not connected');
    }
  };

  const resetTripStatus = () => {
    setTripStatus(null);
    setRequestedTrip(null);
    setActiveTrip(null);
    setPendingCarpoolRequests([]);
  }

  const acceptPendingRequest = (trip: Trip) => {
    if (!driver) return;
    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: { tripID: trip.id, riderID: trip.userID, driver },
    });
    setPendingCarpoolRequests((prev: Trip[]) => prev.filter((t) => t.id !== trip.id));
    if (trip.selectedFare?.packageSlug === CarPackageSlug.CARPOOL) {
      patchDriverSeats(trip.selectedFare?.requestedSeats ?? 1);
    }
  };

  const declinePendingRequest = (trip: Trip) => {
    if (!driver) return;
    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: { tripID: trip.id, riderID: trip.userID, driver, triedDriverIds: triedDriverIdsMap[trip.id] || [] },
    });
    setPendingCarpoolRequests((prev: Trip[]) => prev.filter((t) => t.id !== trip.id));
  };

  const patchDriverSeats = (delta: number) => {
    setDriver((prev: Driver | null) => {
      if (!prev || prev.availableSeats === undefined) return prev;
      return { ...prev, availableSeats: Math.max(0, prev.availableSeats - delta) };
    });
  };

  return { error, tripStatus, driver, requestedTrip, activeTrip, pendingCarpoolRequests, resetTripStatus, sendMessage, setTripStatus, setActiveTrip, patchDriverSeats, acceptPendingRequest, declinePendingRequest, triedDriverIdsMap };
}
