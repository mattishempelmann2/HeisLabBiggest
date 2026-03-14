package elev

import (
	"heis/src/elevio"
)

func (e *Elevator) SteinSaksPapir(Node ElevatorStatus, OtherNodes map[string]ElevatorStatus) { //Utfører steinsakspapir algebra
	for floor := 0; floor < elevio.NumFloors; floor++ {
		for button := 0; button < 2; button++ {
			switch {
			case (e.OrderListHall[floor][button] == Order_Inactive) && (Node.OrderListHall[floor][button] == Order_Pending): // Inaktiv -> Pending
				e.OrderListHall[floor][button] = Order_Pending
			case (e.OrderListHall[floor][button] == Order_Inactive) && Node.OrderListHall[floor][button] == Order_Active: // Inaktiv ->aktiv, skal bare skje ved nettverksbrudd
				e.OrderListHall[floor][button] = Order_Active
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true)
			case (e.OrderListHall[floor][button] == Order_Pending) && ((Node.OrderListHall[floor][button] == Order_Pending) || (Node.OrderListHall[floor][button] == Order_Active)): // Ordre er pending, får enten pending eller aktiv fra annen node -> aktiv
				e.OrderListHall[floor][button] = Order_Active
				e.SetElevButtonLamp(elevio.ButtonType(button), floor, true)
			case (e.OrderListHall[floor][button] == Order_Active) && (Node.OrderListHall[floor][button] == Order_PendingInactive): //Aktiv her, har blitt utført annet sted, gjør klar til å sette utført
				e.OrderListHall[floor][button] = Order_PendingInactive //maybe skru av lys her og

			case (e.OrderListHall[floor][button] == Order_PendingInactive) && Node.OrderListHall[floor][button] == Order_Pending: //ordren var egt utført men ble trykket på nytt, trur ikke denne i praksis vil oppstå, pga speed
				e.OrderListHall[floor][button] = Order_Pending
			case (e.OrderListHall[floor][button] == Order_PendingInactive || e.OrderListHall[floor][button] == Order_Inactive) && (Node.OrderListHall[floor][button] == Order_PendingInactive || Node.OrderListHall[floor][button] == Order_Inactive): // inaktiv/pendingInaktiv her eller på Node
				if e.OrderListHall[floor][button] == Order_PendingInactive { // Er det her den er satt til pendingInaktiv
					ClearConsensus := true                        //Er alle Noder enige, lettere å sjekke etter en negativ, enn å telle antall positive
					for id, otherNodeStatus := range OtherNodes { //iterer liste med status andre noder
						if e.AliveNodes[id] { // Denne checken trengs egentlig ikke da Othernodes i seg selv er en slags AliveNodes, menmen kanskje det trengs down the line
							state := otherNodeStatus.OrderListHall[floor][button]                   //ordren vi sjekker
							if state != Order_Inactive && state != Order_PendingInactive { //hvis ikke inaktiv/PendingInaktiv på alle nodene så er vi ikke klar til å sette til inaktiv
								ClearConsensus = false
								break
							}
						}
					}
					if ClearConsensus {
						e.OrderListHall[floor][button] = Order_Inactive
						e.SetElevButtonLamp(elevio.ButtonType(button), floor, false)
					}
				}
			default: // legge til noe her? Usikker hva default case burde være
				continue
			}
		}
	}
	//skru på lamper cab orders, aktiver de basert på å sjekke map fra andre elev og egen orderlist
	CabBackup, exists := Node.CabBackupMap[e.ID]
	if !exists {
		CabBackup = make([]OrderStatus, elevio.NumFloors) //må manuelt lage dersom det er første gang siden en slice bare returnerer nil dersom ikke eksisterende, fast array returnerer array fyllt med 0.x
	}
	for floor := 0; floor < elevio.NumFloors; floor++ {
		switch {
		case (e.OrderListCab[floor] == Order_Pending) && CabBackup[floor] == Order_Active:
			e.OrderListCab[floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), floor, true)
		case (e.OrderListCab[floor] == Order_Inactive) && CabBackup[floor] == Order_Active && e.MsgCount < 100: // Hvis under 100msg sendt, første sek, oppstart, vi tillater recovery fra andre noder
			if e.Floor == floor && e.DoorOpen { // Unngår dobbel aktivering av ordre i 0 etasje etter reboot, slipper 6 sekund dør åpning
				continue
			}
			e.OrderListCab[floor] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), floor, true)
		default:
			continue
		}
	}
}

func (e *Elevator) CabBackupFunc(Node ElevatorStatus) {
	CabBackup, exists := e.CabBackupMap[Node.SenderID] // Henter map med caborder for NODE vi snakker med atm
	if !exists {
		CabBackup = make([]OrderStatus, elevio.NumFloors)
	}

	for floor := 0; floor < elevio.NumFloors; floor++ { // gjør endringer på map basert på map og melding fra node vi snakker med
		incomingCabstate := Node.OrderListCab[floor]
		currentBackupState := CabBackup[floor]
		switch {
		case (currentBackupState == Order_Inactive) && (incomingCabstate == Order_Pending):
			CabBackup[floor] = Order_Pending

		case (currentBackupState == Order_Pending) && (incomingCabstate == Order_Pending || incomingCabstate == Order_Active):
			CabBackup[floor] = Order_Active

		case (currentBackupState == Order_Active) && (incomingCabstate == Order_Inactive):
			CabBackup[floor] = Order_Inactive

		default:
			continue
		}

	}
	e.CabBackupMap[Node.SenderID] = CabBackup // skriver ny status til map
}
