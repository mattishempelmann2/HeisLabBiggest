package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) UpdateElevatorOrder(btn elevio.ButtonEvent) { //endre btn? bare vi bruker button i resten, men da blir det button.Button
	if e.RunningAlone() {
		if btn.Button < 2 {
			e.OrderListHall[btn.Floor][btn.Button] = Order_Active
		} else {
			e.OrderListCab[btn.Floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), btn.Floor, true)
		}
		return
	}

	if btn.Button < 2 {
		e.OrderListHall[btn.Floor][btn.Button] = Order_Pending
	} else {
		e.OrderListCab[btn.Floor] = Order_Pending
	}
}

func (e *Elevator) HasOrderAbove() bool {
	for floor := e.Floor + 1; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.AssignedOrders[floor][button] || (e.OrderListCab[floor] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) HasOrderBelow() bool {
	for floor := e.Floor - 1; floor >= 0; floor-- {
		for button := 0; button < 2; button++ {
			if e.AssignedOrders[floor][button] || (e.OrderListCab[floor] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) FloorOrder() bool {
	for button := 0; button < 2; button++ {
		if e.AssignedOrders[e.Floor][button] {
			return true
		}
	}
	if e.OrderListCab[e.Floor] == Order_Active {
		return true
	}
	return false
}

func (e *Elevator) ActiveOrders() bool { //needed for PollFloorSensor
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.AssignedOrders[floor][button] || (e.OrderListCab[floor] == Order_Active) {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) ClearOrderFloor() { // mulig ikke lur måte å gjøre det på, rettelse funker clearer i GLOBAL hall orders slik at cost funksjon clearer lokal
	if e.OrderListCab[e.Floor] == Order_Active { //cab orders enkelt og greit
		e.OrderListCab[e.Floor] = Order_Inactive
		e.SetElevButtonLamp(elevio.ButtonType(2), e.Floor, false)
	}

	upAssigned := e.OrderListHall[e.Floor][elevio.BT_HallUp] == Order_Active && e.AssignedOrders[e.Floor][elevio.BT_HallUp]       //Har vi hall ordre oppover
	downAssigned := e.OrderListHall[e.Floor][elevio.BT_HallDown] == Order_Active && e.AssignedOrders[e.Floor][elevio.BT_HallDown] // Har vi hall ordre nedover

	dir := e.Direction         //hva er retning
	if dir == elevio.MD_Stop { // hvis vi står i ro hvordan kom vi hit.
		dir = e.PrevDirection
	}

	clearUp := (dir == elevio.MD_Up) || (dir == elevio.MD_Down && !e.HasOrderBelow()) || (dir == elevio.MD_Stop && e.HasOrderAbove()) //utfør ordre opp om vi var på tur opp/ ned men ingen flere ned/ i ro og skal opp
	clearDown := (dir == elevio.MD_Down) || (dir == elevio.MD_Up && !e.HasOrderAbove()) || (dir == elevio.MD_Stop && e.HasOrderBelow() && !clearUp)

	if dir == elevio.MD_Stop && !clearDown && !clearUp { //edge case ved init, hvor noen kommer inn fra hall, ikke trykket cab order enda.
		if upAssigned {
			clearUp = true
		} else if downAssigned {
			clearDown = true
		}

	}

	if clearUp && upAssigned { //Har vi ordre over og burde vi ta den basert på states
		e.OrderListHall[e.Floor][elevio.BT_HallUp] = Order_PendingInactive
		e.SetElevButtonLamp(elevio.BT_HallUp, e.Floor, false)
		e.AnnouncedDirection = elevio.MD_Up
		return
	}

	if clearDown && downAssigned { //har vi odre ned og burde vi ta den basert på states
		e.OrderListHall[e.Floor][elevio.BT_HallDown] = Order_PendingInactive
		e.SetElevButtonLamp(elevio.BT_HallDown, e.Floor, false)
		e.AnnouncedDirection = elevio.MD_Down
	}
}

func (e *Elevator) UpdateHallLights() {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			if e.OrderListHall[floor][button] == Order_Active {
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true) // holder lys up to date
			} else {
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, false) // skrur av lys etter reset, dersom ordre tatt av annen heis i mellomtiden
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
