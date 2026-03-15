package elev

import (
	"heis/src/elevio"
)

type Orders struct {
	ListHall  [][]OrderStatus
	ListCab   []OrderStatus
	CabBackupList   map[string][]OrderStatus 
	Assigned [][2]bool               
}

type State struct {
	Floor               int
	Direction           elevio.MotorDirection
	PrevDirection       elevio.MotorDirection
	AnnouncedDirection  elevio.MotorDirection
	DoorOpen            bool
	Behaviour           string
	Obstructed          bool
	AnnouncementPending bool
	Stuck               bool
}

type OtherNodes struct {
	Alive map[string]bool
	ID         string
	MessageCount   int
}

type Elevator struct { 
	Orders Orders
	State State
	OtherNodes OtherNodes
}

type ElevatorMessage struct { 
	SenderID     string
	CurrentFloor int
	Direction    int
	DoorOpen     bool
	Behaviour    string

	OrderListHall [][]OrderStatus
	OrderListCab  []OrderStatus 
	CabBackupMap  map[string][]OrderStatus
	MessageID int 
}

type OrderStatus int

const (
	Order_Inactive        = 0 // bruker int, kan eventuelt bruke veldig forskjellieg verdier for å gjøre robust mot "cosmic ray bitflip"
	Order_Pending         = 1 // UDP har vist checksum så mulig irellevant, kanskje bruke 0 til unknown siden det er default value for int?
	Order_Active          = 2
	Order_PendingInactive = 4 // må ha dette eller timestamps
)
