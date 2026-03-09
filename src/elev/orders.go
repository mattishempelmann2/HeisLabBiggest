package elev

import "heis/src/elevio"

func (e *Elevator) UpdateElevatorOrder(btn elevio.ButtonEvent) {
	if btn.Button < 2 {
		e.OrderListHall[btn.Floor][btn.Button] = Order_Pending
	} else {
		e.OrderListCab[btn.Floor] = Order_Pending
	}
}

func (e *Elevator) HasOrderAbove() bool {
	for f := e.Floor + 1; f < elevio.NumFloors; f++ {
		for b := 0; b < 2; b++ {
			if e.AssignedOrders[f][b] || (e.OrderListCab[f] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) HasOrderBelow() bool {
	for f := e.Floor - 1; f >= 0; f-- {
		for b := 0; b < 2; b++ {
			if e.AssignedOrders[f][b] || (e.OrderListCab[f] == Order_Active) {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) FloorOrder() bool {
	for b := 0; b < 2; b++ {
		if e.AssignedOrders[e.Floor][b] {
			return true
		}
	}
	if e.OrderListCab[e.Floor] == Order_Active {
		return true
	}
	return false
}

func (e *Elevator) ActiveOrders() bool { //needed for PollFloorSensor
	for f := 0; f < elevio.NumFloors; f++ {
		for b := 0; b < 2; b++ {
			if e.AssignedOrders[f][b] || (e.OrderListCab[f] == Order_Active) {
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
		return
	}

	if clearDown && downAssigned { //har vi odre ned og burde vi ta den basert på states
		e.OrderListHall[e.Floor][elevio.BT_HallDown] = Order_PendingInactive
		e.SetElevButtonLamp(elevio.BT_HallDown, e.Floor, false)
	}
}

func (e *Elevator) UpdateHallLights() {
	for f := 0; f < elevio.NumFloors; f++ {
		for b := 0; b < 2; b++ {
			if e.OrderListHall[f][b] == Order_Active {
				e.SetElevButtonLamp(elevio.ButtonType(b), f, true) // holder lys up to date
			} else {
				e.SetElevButtonLamp(elevio.ButtonType(b), f, false) // skrur av lys etter reset, dersom ordre tatt av annen heis i mellomtiden
			}

		}
	}
}

func HallOrdersEqual(list1 [][]OrderStatus, list2 [][]OrderStatus) bool {
	if len(list1) != len(list2) {
		return false
	}
	for f := range list1 {
		if len(list1[f]) != len(list2[f]) {
			return false
		}
		for b := range list1[f] {
			if list1[f][b] != list2[f][b] {
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
	for f := range list1 {
		if list1[f] != list2[f] {
			return false
		}
	}
	return true
}
