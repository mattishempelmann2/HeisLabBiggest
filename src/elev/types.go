package elev

import (
	"heis/src/elevio"
)

type Elevator struct {
	OrderListHall  [4][2]OrderStatus
	OrderListCab   [4]OrderStatus
	CabBackupMap   map[string][4]OrderStatus
	AssignedOrders [4][2]bool //orders assigned by costfunk

	Floor         int
	Direction     elevio.MotorDirection
	PrevDirection elevio.MotorDirection
	DoorOpen      bool
	Behaviour     string
	Obstructed    bool

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

	OrderListHall [4][2]OrderStatus
	OrderListCab  [4]OrderStatus
	CabBackupMap  map[string][4]OrderStatus

	MsgID int //For å holde styr på rekkefølge, forkaste gamle meldinger
}

type OrderStatus int

const (
	Order_Inactive = 0 // bruker int, kan eventuelt bruke veldig forskjellieg verdier for å gjøre robust mot "cosmic ray bitflip"
	Order_Pending  = 1 // UDP har vist checksum så mulig irellevant, kanskje bruke 0 til unknown siden det er default value for int?
	Order_Active   = 2
)
