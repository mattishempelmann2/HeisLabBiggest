package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) UpdateElevatorOrder(event elevio.ButtonEvent) {
	if e.RunningAlone() {
		if event.Button < 2 {
			e.Orders.ListHall[event.Floor][event.Button] = Order_Active
		} else {
			e.Orders.ListCab[event.Floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), event.Floor, true)
		}
		return
	}

	if event.Button < 2 {
		e.Orders.ListHall[event.Floor][event.Button] = Order_Pending
	} else {
		e.Orders.ListCab[event.Floor] = Order_Pending
	}
}

func (e *Elevator) HasOrderAbove() bool {
	for floor := e.State.Floor + 1; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.Orders.Assigned[floor][button] || (e.Orders.ListCab[floor] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) HasOrderBelow() bool {
	for floor := e.State.Floor - 1; floor >= 0; floor-- {
		for button := 0; button < 2; button++ {
			if e.Orders.Assigned[floor][button] || (e.Orders.ListCab[floor] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) FloorOrder() bool {
	for button := 0; button < 2; button++ {
		if e.Orders.Assigned[e.State.Floor][button] {
			return true
		}
	}
	if e.Orders.ListCab[e.State.Floor] == Order_Active {
		return true
	}
	return false
}

func (e *Elevator) ActiveOrders() bool {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.Orders.Assigned[floor][button] || (e.Orders.ListCab[floor] == Order_Active) {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) ClearOrderFloor() {
	if e.Orders.ListCab[e.State.Floor] == Order_Active {
		e.Orders.ListCab[e.State.Floor] = Order_Inactive
		e.SetElevButtonLamp(elevio.ButtonType(2), e.State.Floor, false)
	}

	upAssigned := e.Orders.ListHall[e.State.Floor][elevio.BT_HallUp] == Order_Active && e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp]
	downAssigned := e.Orders.ListHall[e.State.Floor][elevio.BT_HallDown] == Order_Active && e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown]

	dir := e.State.Direction
	if dir == elevio.MD_Stop {
		dir = e.State.PrevDirection
	}

	clearUp := (dir == elevio.MD_Up) || (dir == elevio.MD_Down && !e.HasOrderBelow()) || (dir == elevio.MD_Stop && e.HasOrderAbove())
	clearDown := (dir == elevio.MD_Down) || (dir == elevio.MD_Up && !e.HasOrderAbove()) || (dir == elevio.MD_Stop && e.HasOrderBelow() && !clearUp)

	if dir == elevio.MD_Stop && !clearDown && !clearUp { //edge case at init, where someone comes in from hall and hasnt pressed cab order yet.
		if upAssigned {
			clearUp = true
		} else if downAssigned {
			clearDown = true
		}

	}

	if clearUp && upAssigned {
		e.Orders.ListHall[e.State.Floor][elevio.BT_HallUp] = Order_PendingInactive
		e.SetElevButtonLamp(elevio.BT_HallUp, e.State.Floor, false)
		e.State.AnnouncedDirection = elevio.MD_Up
		return
	}

	if clearDown && downAssigned {
		e.Orders.ListHall[e.State.Floor][elevio.BT_HallDown] = Order_PendingInactive
		e.SetElevButtonLamp(elevio.BT_HallDown, e.State.Floor, false)
		e.State.AnnouncedDirection = elevio.MD_Down
	}
}

func (e *Elevator) UpdateHallLights() {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.Orders.ListHall[floor][button] == Order_Active {
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true)
			} else {
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, false)
			}
		}
	}
}

func HallOrdersEqual(list1 [][]OrderStatus, list2 [][]OrderStatus) bool {
	if len(list1) != len(list2) {
		return false
	}
	for floor := range list1 {
		if len(list1[floor]) != len(list2[floor]) {
			return false
		}
		for button := range list1[floor] {
			if list1[floor][button] != list2[floor][button] {
				return false
			}
		}
	}
	return true
}

func CabOrdersEqual(list1 []OrderStatus, list2 []OrderStatus) bool {
	if len(list1) != len(list2) {
		return false
	}
	for floor := range list1 {
		if list1[floor] != list2[floor] {
			return false
		}
	}
	return true
}
