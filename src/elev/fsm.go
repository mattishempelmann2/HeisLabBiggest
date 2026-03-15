package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) StoppFloor() {
	e.SetElevMotorDirection(0)
	e.State.DoorOpen = true
	e.SetElevDoorOpenLamp(true)
	e.ClearOrderFloor()

}

func (e *Elevator) ChooseDirection() elevio.MotorDirection {
	switch e.State.Direction {
	case elevio.MD_Up:
		if e.HasOrderAbove() {
			return elevio.MD_Up
		} else if e.HasOrderBelow() {
			return elevio.MD_Down
		}
		return elevio.MD_Stop
	case elevio.MD_Down:
		if e.HasOrderBelow() {
			return elevio.MD_Down
		} else if e.HasOrderAbove() {
			return elevio.MD_Up
		}
		return elevio.MD_Stop
	case elevio.MD_Stop:
		if e.State.PrevDirection == elevio.MD_Down {
			if e.HasOrderBelow() {
				return elevio.MD_Down
			} else if e.HasOrderAbove() {
				return elevio.MD_Up
			}
		} else {
			if e.HasOrderAbove() {
				return elevio.MD_Up
			} else if e.HasOrderBelow() {
				return elevio.MD_Down
			}
		}
		return elevio.MD_Stop
	default:
		return elevio.MD_Stop
	}
}

func (e *Elevator) ShouldStop() bool {
	if e.Orders.ListCab[e.State.Floor] == Order_Active {
		return true
	}
	dir := e.State.Direction //Forkortelse, skrive helt ut?
	if dir == elevio.MD_Stop {
		dir = e.State.PrevDirection
	}
	switch dir {
	case elevio.MD_Up:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp] || (!e.HasOrderAbove() && e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown])

	case elevio.MD_Down:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown] || (!e.HasOrderBelow() && e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp])

	default:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown] || e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp]
	}

}

func (e *Elevator) ExecuteOrder2() {
	if e.ShouldStop() {
		e.StoppFloor()
		return
	}
	nextDir := e.ChooseDirection()
	e.SetElevMotorDirection(nextDir)
}

func (e *Elevator) RunningAlone() bool {
	for id := range e.OtherNodes.Alive {
		if e.OtherNodes.Alive[id] {
			return false

		}
	}
	return true
}
