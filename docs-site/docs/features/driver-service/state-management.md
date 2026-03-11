---
sidebar_position: 3
title: Driver State Management
---

# Driver State Management

With multiple parallel WebSockets and event pipelines updating driver locations or connecting/disconnecting at random, the Driver Service implements thread-safe state management over the backing memory structures.

## Synchronization (Mutex)

The core `Service` inside `services/driver-service/service.go` maintains an array of pointers to `driverInMap`. To ensure that concurrent gRPC registration calls do not trigger fatal slice panics, `sync.RWMutex` locks the data during writes:

```go
type driverInMap struct {
	Driver *pb.Driver
}

type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex
}
```

## Memory Management

### Connecting (Registration)

When an API Gateway WebSocket connection is upgraded, the remote gRPC client hits `RegisterDriver`. The Service acquires a strict lock, generates simulated variables (Avatar, Plate, Route coordinates), and appends the driver to the active heap:

```go
func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

    // ... populate driver details
	driver := &pb.Driver{
		Id:             driverId,
		PackageSlug:    packageSlug,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	return driver, nil
}
```

### Disconnecting (Cleanup)

When a driver physically closes their laptop or loses cellular connection, the WebSocket in `api-gateway` triggers `defer driverService.Client.UnregisterDriver`. The Driver Service responds by iterating through the memory slice under a strict mutex lock, cleanly slicing out the disconnected driver to prevent "ghost" dispatching:

```go
func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, driver := range s.drivers {
		if driver.Driver.Id == driverId {
			// Slicing out the matched driver
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
		}
	}
}
```

By binding state changes directly to the gRPC streaming contexts derived from the physical WebSocket `defer`, the system efficiently garbage-collects disconnected session data.
