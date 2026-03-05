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
	for f := e.Floor + 1; f < 4; f++ {
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
	for i := 0; i < _numFloors; i++ {
		for j := 0; j < 2; j++ {
			if e.AssignedOrders[i][j] || (e.OrderListCab[i] == Order_Active) {
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

	clearUp := (dir == elevio.MD_Up) || (dir == elevio.MD_Down && !e.HasOrderBelow()) || (dir == elevio.MD_Stop && e.HasOrderAbove()) //utfør ordre opp om vi var på tur opp/ned men ingen flere ned/ i ro og skal opp
	clearDown := (dir == elevio.MD_Down) || (dir == elevio.MD_Up && !e.HasOrderAbove()) || (dir == elevio.MD_Stop && e.HasOrderBelow() && !clearUp)

	if dir == elevio.MD_Stop && !clearDown && !clearUp { //edge case ved init, hvor noen kommer inn fra hall, ikke trykket cab order enda.
		if upAssigned {
			clearUp = true
		} else if downAssigned {
			clearDown = true
		}

	}

	if clearUp && upAssigned {
		e.OrderListHall[e.Floor][elevio.BT_HallUp] = Order_Inactive
		e.SetElevButtonLamp(elevio.BT_HallUp, e.Floor, false)
		return
	}

	if clearDown && downAssigned {
		e.OrderListHall[e.Floor][elevio.BT_HallDown] = Order_Inactive
		e.SetElevButtonLamp(elevio.BT_HallDown, e.Floor, false)
	}
}

func (e *Elevator) UpdateHallLights() {
	for f := 0; f < 4; f++ {
		for b := 0; b < 2; b++ {
			if e.OrderListHall[f][b] == Order_Active {
				e.SetElevButtonLamp(elevio.ButtonType(b), f, true) // holder lys up to date
			} else {
				e.SetElevButtonLamp(elevio.ButtonType(b), f, false) // skrur av lys etter reset, dersom ordre tatt av annen heis i mellomtiden
			}

		}
	}
}
