package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) StoppFloor() {
	e.SetElevMotorDirection(0)
	e.DoorOpen = true
	e.SetElevDoorOpenLamp(true)
	e.ClearOrderFloor()

}

func (e *Elevator) ChooseDirection() elevio.MotorDirection {
	switch e.Direction {
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
		if e.PrevDirection == elevio.MD_Down {
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
	if e.OrderListCab[e.Floor] == Order_Active {
		return true
	}
	dir := e.Direction
	if dir == elevio.MD_Stop {
		dir = e.PrevDirection
	}
	switch dir {
	case elevio.MD_Up:
		return e.AssignedOrders[e.Floor][elevio.BT_HallUp] || (!e.HasOrderAbove() && e.AssignedOrders[e.Floor][elevio.BT_HallDown])

	case elevio.MD_Down:
		return e.AssignedOrders[e.Floor][elevio.BT_HallDown] || (!e.HasOrderBelow() && e.AssignedOrders[e.Floor][elevio.BT_HallUp])

	default:
		return e.AssignedOrders[e.Floor][elevio.BT_HallDown] || e.AssignedOrders[e.Floor][elevio.BT_HallUp]
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
