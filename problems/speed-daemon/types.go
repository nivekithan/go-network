package main

type ClientError struct {
	msg string
}

type Plate struct {
	plate     string
	timestamp uint32
}

type Ticket struct {
	plate      string
	road       uint16
	mile1      uint16
	timestamp1 uint32
	mile2      uint16
	timestamp2 uint32

	// Speed is represented by 100 * miles / hour
	speed uint16
}

type WantHeartbeat struct {
	// Interval is represented in deciseconds
	// 25 deciseconds = 2.5 seconds
	interval uint32
}

type Heartbeat struct {
}

type IAmCamera struct {
	road  uint16
	mile  uint16
	limit uint16
}

type IamDispatcher struct {
	numRoads uint8
	roads    []uint16
}
