package elev

import (
	"heis/src/elevio"
	"time"
)

const _pollRate = 20 * time.Millisecond

func (e *Elevator) UpdateFloor(Floor int) {
	if Floor != -1 {
		e.State.Floor = Floor
	}
}

func (e *Elevator) UpdateDirection(Direction elevio.MotorDirection) {
	e.State.PrevDirection = e.State.Direction
	e.State.Direction = Direction
}

func (e *Elevator) CabInit(ID string, numFloors int) {
	e.Orders.ListHall = make([][]OrderStatus, numFloors) //slice lager numfloors antall som igjen inneholder liste med Orderstatus
	e.Orders.Assigned = make([][2]bool, numFloors)       //Skal vi gjøre denne dynamisk?
	for floor := range e.Orders.ListHall {
		e.Orders.ListHall[floor] = make([]OrderStatus, 2) //fyller for hver etasje, antall knapper er fixed
	}
	e.Orders.ListCab = make([]OrderStatus, numFloors)
	e.Orders.CabBackupList = make(map[string][]OrderStatus)

	for elevio.GetFloor() != 0 { //kjør ned til bunn
		e.SetElevMotorDirection(elevio.MD_Down)
		time.Sleep(_pollRate)
	}
	e.SetElevMotorDirection(elevio.MD_Stop)

	e.State.Floor = 0                      //nulte etasje
	e.State.PrevDirection = elevio.MD_Stop //sist retning
	e.State.Direction = elevio.MD_Stop     //beveger seg ikke
	e.State.DoorOpen = false
	e.SetElevDoorOpenLamp(false)
	e.OtherNodes.Alive = make(map[string]bool)
	e.OtherNodes.ID = ID
	e.OtherNodes.MessageCount = 0
	e.State.Obstructed = false
	e.State.Stuck = false
}

func (e *Elevator) UpdateBehaviour() {
	switch {
	case e.State.DoorOpen:
		e.State.Behaviour = "doorOpen"
	case e.State.Direction != 0:
		e.State.Behaviour = "moving"
	default:
		e.State.Behaviour = "idle"
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
