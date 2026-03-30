import { Polyline } from "react-leaflet";
import { Route } from "../types";

export function RoutingControl({ route }: {
    route: Route
}) {
    if (!route || !route.geometry) return null;

    // Different colors for each leg of the trip to highlight overlaps
    const colors = ["#3388ff", "#9c27b0", "#ff9800", "#e91e63", "#4caf50"];

    return (
        <>
            {route.geometry.map((geo, index) => {
                const positions = geo.coordinates.map(c => [c.latitude, c.longitude] as [number, number]);
                const color = colors[index % colors.length];
                return (
                    <Polyline 
                        key={index} 
                        positions={positions} 
                        color={color} 
                        weight={index % 2 !== 0 ? 6 : 4} // highlight alternate segments slightly thicker
                    />
                );
            })}
        </>
    );
}