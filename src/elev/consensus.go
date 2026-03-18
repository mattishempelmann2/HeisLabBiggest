package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) HallConsensus(Node ElevatorMessage, OtherNodes map[string]ElevatorMessage) {
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			switch {
			case (e.Orders.ListHall[floor][button] == Order_Inactive) && (Node.OrderListHall[floor][button] == Order_Pending): // Inactive -> Pending
				e.Orders.ListHall[floor][button] = Order_Pending

			case (e.Orders.ListHall[floor][button] == Order_Inactive) && Node.OrderListHall[floor][button] == Order_Active: // Inactive ->active, should only happen after network loss
				e.Orders.ListHall[floor][button] = Order_Active
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true)

			case (e.Orders.ListHall[floor][button] == Order_Pending) && ((Node.OrderListHall[floor][button] == Order_Pending) || (Node.OrderListHall[floor][button] == Order_Active)): // Order is pending, recieves either pending or active -> active
				e.Orders.ListHall[floor][button] = Order_Active
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true)

			case (e.Orders.ListHall[floor][button] == Order_Active) && (Node.OrderListHall[floor][button] == Order_PendingInactive): //Active here, has been executed somewhere else -> pending ianactive
				e.Orders.ListHall[floor][button] = Order_PendingInactive

			case (e.Orders.ListHall[floor][button] == Order_PendingInactive) && Node.OrderListHall[floor][button] == Order_Pending: //Order was executed but was reordered
				e.Orders.ListHall[floor][button] = Order_Pending

			case (e.Orders.ListHall[floor][button] == Order_PendingInactive || e.Orders.ListHall[floor][button] == Order_Inactive) && (Node.OrderListHall[floor][button] == Order_PendingInactive || Node.OrderListHall[floor][button] == Order_Inactive): // inactive/pendingInactive here or on external Node
				if e.Orders.ListHall[floor][button] == Order_PendingInactive {
					ClearConsensus := true
					for id, otherNodeStatus := range OtherNodes {
						if e.OtherNodes.Alive[id] {
							states := otherNodeStatus.OrderListHall[floor][button]
							if states != Order_Inactive && states != Order_PendingInactive { //if not in agreement, then we are not ready to set inactive.
								ClearConsensus = false
								break
							}
						}
					}
					if ClearConsensus {
						e.Orders.ListHall[floor][button] = Order_Inactive
						e.SetElevButtonLamp(elevio.ButtonType(button), floor, false)
					}
				}
			default:
				continue
			}
		}
	}

	CabBackup, exists := Node.CabBackupMap[e.OtherNodes.ID]
	if !exists {
		CabBackup = make([]OrderStatus, elevio.NumFloors)
	}
	for floor := 0; floor < elevio.NumFloors; floor++ {
		switch {
		case (e.Orders.ListCab[floor] == Order_Pending) && CabBackup[floor] == Order_Active:
			e.Orders.ListCab[floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), floor, true)
		case (e.Orders.ListCab[floor] == Order_Inactive) && CabBackup[floor] == Order_Active && e.OtherNodes.MessageCount < 100: //If under 100 messages sent, recovery of caborders is active.
			if e.State.Floor == floor && e.State.DoorOpen { //Handles edge case to avoid double opening of door in floor 0 after reboot.
				continue
			}
			e.Orders.ListCab[floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), floor, true)
		default:
			continue
		}
	}
}

func (e *Elevator) CabBackupFunc(Node ElevatorMessage) {
	cabBackup, exists := e.Orders.CabBackupList[Node.SenderID]
	if !exists {
		cabBackup = make([]OrderStatus, elevio.NumFloors)
	}

	for floor := 0; floor < elevio.NumFloors; floor++ {
		incomingCabStates := Node.OrderListCab[floor]
		currentBackupStates := cabBackup[floor]
		switch {
		case (currentBackupStates == Order_Inactive) && (incomingCabStates == Order_Pending):
			cabBackup[floor] = Order_Pending

		case (currentBackupStates == Order_Pending) && (incomingCabStates == Order_Pending || incomingCabStates == Order_Active):
			cabBackup[floor] = Order_Active

		case (currentBackupStates == Order_Active) && (incomingCabStates == Order_Inactive):
			cabBackup[floor] = Order_Inactive

		default:
			continue
		}

	}
	e.Orders.CabBackupList[Node.SenderID] = cabBackup // Writes new status to map
}
