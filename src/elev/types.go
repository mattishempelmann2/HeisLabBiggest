package elev

import (
	"heis/src/elevio"
)

type Elevator struct { //split opp i mindre structs
	OrderListHall  [][]OrderStatus
	OrderListCab   []OrderStatus
	CabBackupMap   map[string][]OrderStatus //nå slices så dynamisk lengde
	AssignedOrders [][2]bool                //orders assigned by costfunk

	Floor               int
	Direction           elevio.MotorDirection
	PrevDirection       elevio.MotorDirection
	AnnouncedDirection  elevio.MotorDirection
	DoorOpen            bool
	Behaviour           string
	Obstructed          bool
	AnnouncementPending bool
	Stuck               bool

	AliveNodes map[string]bool
	ID         string
	MsgCount   int
}

type ElevatorStatus struct { //det som sendes, health checks
	SenderID     string
	CurrentFloor int
	Direction    int
	DoorOpen     bool
	Behaviour    string

	OrderListHall [][]OrderStatus
	OrderListCab  []OrderStatus //slices, dynamisk lengde
	CabBackupMap  map[string][]OrderStatus

	MsgID int //For å holde styr på rekkefølge, forkaste gamle meldinger
}

type OrderStatus int

const (
	Order_Inactive        = 0 // bruker int, kan eventuelt bruke veldig forskjellieg verdier for å gjøre robust mot "cosmic ray bitflip"
	Order_Pending         = 1 // UDP har vist checksum så mulig irellevant, kanskje bruke 0 til unknown siden det er default value for int?
	Order_Active          = 2
	Order_PendingInactive = 4 // må ha dette eller timestamps
)
