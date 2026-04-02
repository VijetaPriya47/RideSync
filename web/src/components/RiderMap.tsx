/* eslint-disable */
'use client';

import Image from 'next/image';
import { useRiderStreamConnection } from '../hooks/useRiderStreamConnection';
import { MapContainer, Marker, Popup, Rectangle, TileLayer, useMap } from 'react-leaflet'
import L from 'leaflet';
import { getGeohashBounds } from '../utils/geohash';
import { useMemo, useRef, useState, useEffect } from 'react';
import { MapClickHandler } from './MapClickHandler';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Search, MapPin, Navigation, History, Loader2 } from 'lucide-react';
import { RouteFare, RequestRideProps, TripPreview, HTTPTripStartResponse } from "../types";
import { RoutingControl } from "./RoutingControl";
import { API_URL } from '../constants';
import { RiderTripOverview } from './RiderTripOverview';
import { BackendEndpoints, HTTPTripPreviewRequestPayload, HTTPTripPreviewResponse, HTTPTripStartRequestPayload, TripEvents } from '../contracts';

const carSvg = encodeURIComponent(`
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#000000" stroke="#ffffff" stroke-width="0.5">
  <path d="M18.92 6.01C18.72 5.42 18.16 5 17.5 5h-11c-.66 0-1.21.42-1.42 1.01L3 12v8c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-1h12v1c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-8l-2.08-5.99zM6.5 16c-.83 0-1.5-.67-1.5-1.5S5.67 13 6.5 13s1.5.67 1.5 1.5S7.33 16 6.5 16zm11 0c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5zM5 11l1.5-4.5h11L19 11H5z"/>
</svg>
`);

const userMarker = L.divIcon({
    className: 'custom-pickup-marker',
    html: `
        <div style="position:relative; width:24px; height:24px; display:flex; align-items:center; justify-content:center;">
            <div style="width:12px; height:12px; background-color:#2563eb; border-radius:50%; border:2px solid white; box-shadow:0 0 6px rgba(0,0,0,0.4); z-index:2;"></div>
        </div>
    `,
    iconSize: [24, 24],
    iconAnchor: [12, 12]
});

const destinationMarker = L.divIcon({
    className: 'custom-dest-marker',
    html: `<div style="width:14px; height:14px; background-color:#111827; border:2px solid white; box-shadow:0 0 6px rgba(0,0,0,0.4);"></div>`,
    iconSize: [14, 14],
    iconAnchor: [7, 7]
});

const driverMarker = new L.Icon({
    iconUrl: `data:image/svg+xml;utf8,${carSvg}`,
    iconSize: [36, 36],
    iconAnchor: [18, 18],
});

interface RiderMapProps {
    onRouteSelected?: (distance: number) => void;
}

const DelhiCenter = {
    latitude: 28.6139,
    longitude: 77.2090,
};

// Component to handle map animation
function MapMover({ center, zoom }: { center: [number, number], zoom: number }) {
    const map = useMap();
    useEffect(() => {
        map.setView(center, zoom, { animate: true });
        map.invalidateSize(); // Force redraw to fix centering issues
    }, [center, zoom, map]);
    return null;
}

export default function RiderMap({ onRouteSelected }: RiderMapProps) {
    const [trip, setTrip] = useState<TripPreview | null>(null)
    const [selectedCarPackage, setSelectedCarPackage] = useState<RouteFare | null>(null)
    const [destination, setDestination] = useState<[number, number] | null>(null)
    const [location, setLocation] = useState(DelhiCenter)
    const [search, setSearch] = useState({ pickup: '', destination: '' })
    const [suggestions, setSuggestions] = useState<{ pickup: any[], destination: any[] }>({ pickup: [], destination: [] })
    const [isSearching, setIsSearching] = useState({ pickup: false, destination: false })
    const [isGPSTracking, setIsGPSTracking] = useState(false)
    const [gpsAvailable, setGpsAvailable] = useState(true)

    const mapRef = useRef<L.Map>(null)
    const userID = useMemo(() => crypto.randomUUID(), [])
    const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const {
        drivers,
        error,
        tripStatus,
        assignedDriver,
        paymentSession,
        resetTripStatus,
        setTripStatus
    } = useRiderStreamConnection(location, userID);

    // Auto-center on user's real location once on mount
    useEffect(() => {
        if (!navigator.geolocation) { setGpsAvailable(false); return; }
        navigator.geolocation.getCurrentPosition(
            (pos) => {
                setLocation({ latitude: pos.coords.latitude, longitude: pos.coords.longitude });
            },
            () => setGpsAvailable(false),
            { enableHighAccuracy: true, timeout: 8000 }
        );
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Live GPS tracking when enabled
    useEffect(() => {
        if (!isGPSTracking) return;
        const watchId = navigator.geolocation.watchPosition(
            (pos) => {
                const coords = { latitude: pos.coords.latitude, longitude: pos.coords.longitude };
                setLocation(coords);
                setSearch(prev => ({ ...prev, pickup: 'My Location (GPS)' }));
            },
            () => { setIsGPSTracking(false); },
            { enableHighAccuracy: true }
        );
        return () => navigator.geolocation.clearWatch(watchId);
    }, [isGPSTracking]);

    const handleSearch = (query: string, type: 'pickup' | 'destination') => {
        setSearch(prev => ({ ...prev, [type]: query }));

        if (debounceTimeoutRef.current) clearTimeout(debounceTimeoutRef.current);

        if (query.length < 3) {
            setSuggestions(prev => ({ ...prev, [type]: [] }));
            return;
        }

        debounceTimeoutRef.current = setTimeout(async () => {
            setIsSearching(prev => ({ ...prev, [type]: true }));
            try {
                // Biased search towards New Delhi via viewbox (left, top, right, bottom)
                const resp = await fetch(`https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(query)}&limit=5&viewbox=76.8,28.9,77.4,28.4&countrycodes=in`);
                const data = await resp.json();
                setSuggestions(prev => ({ ...prev, [type]: data }));
            } catch (e) {
                console.error("Geocoding failed", e);
            } finally {
                setIsSearching(prev => ({ ...prev, [type]: false }));
            }
        }, 500);
    }

    const selectSuggestion = (s: any, type: 'pickup' | 'destination') => {
        const lat = parseFloat(s.lat);
        const lon = parseFloat(s.lon);

        if (type === 'pickup') {
            setLocation({ latitude: lat, longitude: lon });
            setSearch(prev => ({ ...prev, pickup: s.display_name }));
        } else {
            setDestination([lat, lon]);
            setSearch(prev => ({ ...prev, destination: s.display_name }));
            // Trigger preview calculation automatically
            handleAddressSelection(lat, lon);
        }
        setSuggestions(prev => ({ ...prev, [type]: [] }));
    }

    const handleAddressSelection = async (destLat: number, destLon: number) => {
        const data = await requestRidePreview({
            pickup: [location.latitude, location.longitude],
            destination: [destLat, destLon],
        })
        setTrip({
            tripID: "",
            route: data.route,
            rideFares: data.rideFares,
            distance: data.route.distance,
            duration: data.route.duration,
        })
    }

    const requestRidePreview = async (req: RequestRideProps) => {
        const payload = {
            userID,
            pickup: {
                latitude: req.pickup[0],
                longitude: req.pickup[1],
            },
            destination: {
                latitude: req.destination[0],
                longitude: req.destination[1],
            },
            requestedSeats: 1,
        } as HTTPTripPreviewRequestPayload

        const response = await fetch(`${API_URL}${BackendEndpoints.PREVIEW_TRIP}`, {
            method: 'POST',
            body: JSON.stringify(payload),
        })
        const { data } = await response.json() as { data: HTTPTripPreviewResponse }
        return data
    }

    const handleMapClick = async (e: L.LeafletMouseEvent) => {
        if (trip?.tripID) return;
        setDestination([e.latlng.lat, e.latlng.lng]);
        setSearch(prev => ({ ...prev, destination: 'Point on Map' }));
        const data = await requestRidePreview({
            pickup: [location.latitude, location.longitude],
            destination: [e.latlng.lat, e.latlng.lng],
        })
        setTrip({
            tripID: "",
            route: data.route,
            rideFares: data.rideFares,
            distance: data.route.distance,
            duration: data.route.duration,
        })

        if (onRouteSelected) {
            onRouteSelected(data.route.distance)
        }
    }

    const handleStartTrip = async (fare: RouteFare) => {
        if (fare.packageSlug === 'carpool' && !fare.requestedSeats) {
            fare.requestedSeats = 1;
        }
        setSelectedCarPackage(fare)
        const payload = {
            rideFareID: fare.id,
            userID: userID,
        } as HTTPTripStartRequestPayload

        const response = await fetch(`${API_URL}${BackendEndpoints.START_TRIP}`, {
            method: 'POST',
            body: JSON.stringify(payload),
        })
        const { data } = await response.json() as { data: HTTPTripStartResponse }

        if (response.ok && trip) {
            setTrip((prev) => ({
                ...prev,
                tripID: data.tripID,
            } as TripPreview))
        }
        return data
    }

    const handleCancelTrip = () => {
        setTrip(null)
        setDestination(null)
        setTripStatus(null)
        setSelectedCarPackage(null)
        setSearch({ pickup: '', destination: '' })
    }

    const handleIncreaseFare = async (percentage: number) => {
        if (!trip?.tripID || !selectedCarPackage?.totalPriceInCents) return;

        try {
            const url = `${API_URL}${BackendEndpoints.GET_TRIP}`.replace('{id}', trip.tripID);
            const statusResp = await fetch(url);
            if (statusResp.ok) {
                const { data } = await statusResp.json();
                if (data.status === 'accepted' || data.status === 'completed' || data.status === 'cancelled') {
                    alert(`Cannot increase fare. Trip is already ${data.status}.`);
                    return;
                }
            }
        } catch (e) {
            console.error("Failed to check trip status", e);
        }

        const newPrice = selectedCarPackage.totalPriceInCents * (1 + percentage / 100);
        const payload = { tripID: trip.tripID, userID: userID, totalPriceInCents: newPrice };
        const response = await fetch(`${API_URL}${BackendEndpoints.INCREASE_TRIP_FARE}`, {
            method: 'POST',
            body: JSON.stringify(payload),
            headers: { 'Content-Type': 'application/json' }
        });
        if (response.ok) {
            setSelectedCarPackage({ ...selectedCarPackage, totalPriceInCents: newPrice });
            setTripStatus(TripEvents.Created);
        } else {
            alert("Failed to increase fare");
        }
    };

    return (
        <div className="relative flex flex-col md:flex-row h-screen font-sans">
            <div className="flex-1 relative">
                {/* Search Panel Overlay */}
                <div className="absolute top-4 left-4 right-4 z-[9999] max-w-md mx-auto md:mx-0">
                    <div className="bg-white/95 backdrop-blur-sm rounded-2xl shadow-xl border border-gray-100 p-3 flex flex-col gap-2">
                        <div className="relative flex items-center group">
                            <div className="absolute left-3 text-blue-500">
                                <Navigation className="w-4 h-4" />
                            </div>
                            <Input
                                placeholder="Enter Pickup Location..."
                                value={search.pickup}
                                onChange={(e) => handleSearch(e.target.value, 'pickup')}
                                className="pl-10 h-11 bg-gray-50/50 border-none rounded-xl focus-visible:ring-blue-500"
                            />
                            {isSearching.pickup && <Loader2 className="absolute right-3 w-4 h-4 animate-spin text-gray-400" />}
                            {suggestions.pickup.length > 0 && (
                                <div className="absolute top-full left-0 right-0 mt-2 bg-white rounded-xl shadow-2xl border border-gray-100 overflow-hidden z-[10000]">
                                    {suggestions.pickup.map((s, i) => (
                                        <button
                                            key={i}
                                            onClick={() => selectSuggestion(s, 'pickup')}
                                            className="w-full px-4 py-3 text-left text-sm hover:bg-blue-50 border-b border-gray-50 last:border-none flex items-center gap-3"
                                        >
                                            <MapPin className="w-4 h-4 text-gray-400" />
                                            <span className="truncate">{s.display_name}</span>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>

                        <div className="relative flex items-center group">
                            <div className="absolute left-3 text-red-500">
                                <MapPin className="w-4 h-4" />
                            </div>
                            <Input
                                placeholder="Where to?"
                                value={search.destination}
                                onChange={(e) => handleSearch(e.target.value, 'destination')}
                                className="pl-10 h-11 bg-gray-50/50 border-none rounded-xl focus-visible:ring-red-500"
                            />
                            {isSearching.destination && <Loader2 className="absolute right-3 w-4 h-4 animate-spin text-gray-400" />}
                            {suggestions.destination.length > 0 && (
                                <div className="absolute top-full left-0 right-0 mt-2 bg-white rounded-xl shadow-2xl border border-gray-100 overflow-hidden z-[10000]">
                                    {suggestions.destination.map((s, i) => (
                                        <button
                                            key={i}
                                            onClick={() => selectSuggestion(s, 'destination')}
                                            className="w-full px-4 py-3 text-left text-sm hover:bg-blue-50 border-b border-gray-50 last:border-none flex items-center gap-3"
                                        >
                                            <History className="w-4 h-4 text-gray-400" />
                                            <span className="truncate">{s.display_name}</span>
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                <MapContainer
                    center={[location.latitude, location.longitude]}
                    zoom={13}
                    style={{ height: '100%', width: '100%' }}
                    ref={mapRef}
                >
                    <TileLayer
                        url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                        attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors"
                    />
                    <MapMover center={[location.latitude, location.longitude]} zoom={14} />

                    <Marker position={[location.latitude, location.longitude]} icon={userMarker} />

                    {drivers?.map((driver) => (
                        <Marker
                            key={driver?.id}
                            position={[driver?.location?.latitude, driver?.location?.longitude]}
                            icon={driverMarker}
                        >
                            <Popup>
                                <div className="flex flex-col gap-1">
                                    <span className="font-bold">{driver?.name}</span>
                                    <span className="text-xs text-gray-500">{driver?.carPlate}</span>
                                </div>
                            </Popup>
                        </Marker>
                    ))}

                    {destination && (
                        <Marker position={destination} icon={destinationMarker}>
                            <Popup>Destination</Popup>
                        </Marker>
                    )}

                    {trip?.route && (
                        <RoutingControl route={trip.route} />
                    )}
                    <MapClickHandler onClick={handleMapClick} />
                </MapContainer>

                {/* Google Maps-style My Location FAB */}
                {gpsAvailable && (
                    <button
                        onClick={() => setIsGPSTracking(prev => !prev)}
                        title={isGPSTracking ? 'GPS Tracking Active' : 'Center on my location'}
                        className="absolute bottom-6 right-4 z-[1000] w-10 h-10 bg-white rounded-full shadow-[0_1px_4px_rgba(0,0,0,0.3)] border border-gray-200 flex items-center justify-center hover:bg-gray-50 active:scale-95 transition-all duration-150"
                    >
                        {isGPSTracking ? (
                            /* Filled blue dot — active GPS */
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="20" height="20">
                                <circle cx="12" cy="12" r="8" fill="#1A73E8" opacity="0.2" />
                                <circle cx="12" cy="12" r="4" fill="#1A73E8" />
                                <line x1="12" y1="2" x2="12" y2="6" stroke="#1A73E8" strokeWidth="2" strokeLinecap="round" />
                                <line x1="12" y1="18" x2="12" y2="22" stroke="#1A73E8" strokeWidth="2" strokeLinecap="round" />
                                <line x1="2" y1="12" x2="6" y2="12" stroke="#1A73E8" strokeWidth="2" strokeLinecap="round" />
                                <line x1="18" y1="12" x2="22" y2="12" stroke="#1A73E8" strokeWidth="2" strokeLinecap="round" />
                            </svg>
                        ) : (
                            /* Outlined crosshair — idle */
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="20" height="20">
                                <circle cx="12" cy="12" r="7" fill="none" stroke="#5F6368" strokeWidth="2" />
                                <circle cx="12" cy="12" r="2" fill="#5F6368" />
                                <line x1="12" y1="2" x2="12" y2="5" stroke="#5F6368" strokeWidth="2" strokeLinecap="round" />
                                <line x1="12" y1="19" x2="12" y2="22" stroke="#5F6368" strokeWidth="2" strokeLinecap="round" />
                                <line x1="2" y1="12" x2="5" y2="12" stroke="#5F6368" strokeWidth="2" strokeLinecap="round" />
                                <line x1="19" y1="12" x2="22" y2="12" stroke="#5F6368" strokeWidth="2" strokeLinecap="round" />
                            </svg>
                        )}
                    </button>
                )}
            </div>

            <div className="flex flex-col w-full md:w-[450px] bg-white border-l border-gray-100">
                <RiderTripOverview
                    trip={trip}
                    selectedFare={selectedCarPackage}
                    assignedDriver={assignedDriver}
                    status={tripStatus}
                    paymentSession={paymentSession}
                    onPackageSelect={handleStartTrip}
                    onCancel={handleCancelTrip}
                    onIncreaseFare={handleIncreaseFare}
                />
            </div>
        </div>
    )
}