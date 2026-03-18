package elev

import (
	"heis/src/elevio"
)

type Orders struct {
	ListHall      [][]OrderStatus
	ListCab       []OrderStatus
	CabBackupList map[string][]OrderStatus
	Assigned      [][2]bool
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
	Alive        map[string]bool
	ID           string
	MessageCount int
}

type Elevator struct {
	Orders     Orders
	State      State
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
	MessageID     int
}

type OrderStatus int

const (
	Order_Inactive        = 0
	Order_Pending         = 1
	Order_Active          = 2
	Order_PendingInactive = 4
)

var dirMap = map[int]string{
	1:  "up",
	-1: "down",
	0:  "stop",
}
