import { Coordinate, Driver, Route, RouteFare, Trip } from "./types";

// These are the endpoints the API Gateway must have for the frontend to work correctly
export enum BackendEndpoints {
  PREVIEW_TRIP = "/trip/preview",
  START_TRIP = "/trip/start",
  GET_TRIP = "/trip/{id}",
  INCREASE_TRIP_FARE = "/trip/increase-fare",
  UPDATE_TRIP_SEATS = "/trip/update-seats",
  WS_DRIVERS = "/drivers",
  WS_RIDERS = "/riders",
}

export enum TripEvents {
  NoDriversFound = "trip.event.no_drivers_found",
  DriverAssigned = "trip.event.driver_assigned",
  Completed = "trip.event.completed",
  Cancelled = "trip.event.cancelled",
  Created = "trip.event.created",
  DriverLocation = "driver.cmd.location",
  DriverTripRequest = "driver.cmd.trip_request",
  DriverTripAccept = "driver.cmd.trip_accept",
  DriverTripDecline = "driver.cmd.trip_decline",
  DriverRegister = "driver.cmd.register",
  PaymentSessionCreated = "payment.event.session_created",
}

// Messages sent from the server to the client via the websocket
export type ServerWsMessage =
  | PaymentSessionCreatedRequest
  | DriverAssignedRequest
  | DriverLocationRequest
  | DriverTripRequest
  | DriverRegisterRequest
  | TripCreatedRequest
  | NoDriversFoundRequest;

// Messages sent from the client to the server via the websocket
export type ClientWsMessage = DriverResponseToTripResponse

interface TripCreatedRequest {
  type: TripEvents.Created;
  data: Trip;
}

interface NoDriversFoundRequest {
  type: TripEvents.NoDriversFound;
}

interface DriverRegisterRequest {
  type: TripEvents.DriverRegister;
  data: Driver;
}
interface DriverTripRequest {
  type: TripEvents.DriverTripRequest;
  data: Trip;
}

export interface PaymentEventSessionCreatedData {
  tripID: string;
  sessionID: string;
  amount: number;
  currency: string;
}

interface PaymentSessionCreatedRequest {
  type: TripEvents.PaymentSessionCreated;
  data: PaymentEventSessionCreatedData;
}

interface DriverAssignedRequest {
  type: TripEvents.DriverAssigned;
  data: Trip;
}

interface DriverLocationRequest {
  type: TripEvents.DriverLocation;
  data: Driver[];
}

interface DriverResponseToTripResponse {
  type: TripEvents.DriverTripAccept | TripEvents.DriverTripDecline;
  data: {
    tripID: string;
    riderID: string;
    driver: Driver;
    triedDriverIds?: string[];
  };
}

export interface HTTPTripPreviewResponse {
  route: Route;
  rideFares: RouteFare[];
}

export interface HTTPTripStartRequestPayload {
  rideFareID: string;
  /** Omit so the gateway sets user id from the JWT (avoids stale localStorage mismatch). */
  userID?: string;
}

export interface HTTPTripPreviewRequestPayload {
  /** Omit so the gateway sets user id from the JWT (avoids stale localStorage mismatch). */
  userID?: string;
  pickup: Coordinate;
  destination: Coordinate;
  /** Seats for carpool-style fares; defaults to 1 if omitted */
  requestedSeats?: number;
}

export function isValidTripEvent(event: string): event is TripEvents {
  return Object.values(TripEvents).includes(event as TripEvents);
}

export function isValidWsMessage(message: ServerWsMessage): message is ServerWsMessage {
  return isValidTripEvent(message.type);
}
