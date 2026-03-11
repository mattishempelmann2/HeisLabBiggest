package elev

import (
	"heis/src/elevio"
	"time"
)

const _pollRate = 20 * time.Millisecond

func (e *Elevator) UpdateFloor(Floor int) {
	if Floor != -1 {
		e.Floor = Floor
	}
}

func (e *Elevator) UpdateDirection(Direction elevio.MotorDirection) {
	e.PrevDirection = e.Direction
	e.Direction = Direction
}

func (e *Elevator) CabInit(ID string, numFloors int) {
	e.OrderListHall = make([][]OrderStatus, numFloors) //slice lager numfloors antall som igjen inneholder liste med Orderstatus
	e.AssignedOrders = make([][2]bool, numFloors)
	for i := range e.OrderListHall {
		e.OrderListHall[i] = make([]OrderStatus, 2) //fyller for hver etasje, antall knapper er fixed
	}
	e.OrderListCab = make([]OrderStatus, numFloors)
	e.CabBackupMap = make(map[string][]OrderStatus)

	for elevio.GetFloor() != 0 { //kjør ned til bunn
		e.SetElevMotorDirection(elevio.MD_Down)
		time.Sleep(_pollRate)
	}
	e.SetElevMotorDirection(elevio.MD_Stop)

	e.Floor = 0
	e.PrevDirection = elevio.MD_Stop
	e.Direction = elevio.MD_Stop
	e.DoorOpen = false
	e.SetElevDoorOpenLamp(false)
	e.AliveNodes = make(map[string]bool)
	e.ID = ID
	e.MsgCount = 0
	e.Obstructed = false
	e.Stuck = false
}

func (e *Elevator) UpdateBehaviour() {
	switch {
	case e.DoorOpen:
		e.Behaviour = "doorOpen"
	case e.Direction != 0:
		e.Behaviour = "moving"
	default:
		e.Behaviour = "idle"
	}
}

func (e *Elevator) SetElevMotorDirection(dir elevio.MotorDirection) {
	elevio.SetMotorDirection(dir)
	e.UpdateDirection(dir)
	e.UpdateBehaviour()
}

func (e *Elevator) SetElevButtonLamp(button elevio.ButtonType, floor int, value bool) {
	elevio.SetButtonLamp(button, floor, value)
}

func (e *Elevator) SetElevDoorOpenLamp(value bool) {
	elevio.SetDoorOpenLamp(value)
	e.UpdateBehaviour()
}
