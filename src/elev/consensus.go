package elev

import (
	"heis/src/elevio"
)

var _numFloors int = 4

func (e *Elevator) SteinSaksPapir(Node ElevatorStatus) { //Utfører steinsakspapir algebra
	for i := 0; i < _numFloors; i++ {
		for j := 0; j < 2; j++ {
			switch {
			case (e.OrderListHall[i][j] == Order_Inactive) && (Node.OrderListHall[i][j] == Order_Pending): // var inaktiv, får pending fra annen node = pending
				e.OrderListHall[i][j] = Order_Pending
			case (e.OrderListHall[i][j] == Order_Pending) && ((Node.OrderListHall[i][j] == Order_Pending) || (Node.OrderListHall[i][j] == Order_Active)): // Ordre er pending, får enten pending eller aktiv fra annen node -> aktiv
				e.OrderListHall[i][j] = Order_Active
				e.SetElevButtonLamp(elevio.ButtonType(j), i, true) // noe av det dummeste jeg har sett, caste i som er en int til buttontype som er en int
			case (e.OrderListHall[i][j] == Order_Active) && (Node.OrderListHall[i][j] == Order_Inactive): // Ordre er aktiv, blir utført annen node->satt inaktiv der = inaktiv her
				e.OrderListHall[i][j] = Order_Inactive
				e.SetElevButtonLamp(elevio.ButtonType(j), i, false)
			default: // legge til noe her? Usikker hva default case burde være
				continue
			}
		}
	}
	//skru på lamper cab orders, aktiver de basert på å sjekke map fra andre elev og egen orderlist
	CabBackup := Node.CabBackupMap[e.ID]
	for k := 0; k < _numFloors; k++ {
		switch {
		case (e.OrderListCab[k] == Order_Pending) && CabBackup[k] == Order_Active:
			e.OrderListCab[k] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), k, true)
		case (e.OrderListCab[k] == Order_Inactive) && CabBackup[k] == Order_Active && e.MsgCount < 100: // Hvis under 100msg sendt, første sek, oppstart, vi tillater recovery fra andre noder
			if e.Floor == k && e.DoorOpen { // Unngår dobbel aktivering av ordre i 0 etasje etter reboot, slipper 6 sekund dør åpning
				continue
			}
			e.OrderListCab[k] = Order_Active
			e.SetElevButtonLamp(elevio.ButtonType(2), k, true)
		default:
			continue
		}
	}
}

func (e *Elevator) CabBackupFunc(Node ElevatorStatus) {
	CabBackup := e.CabBackupMap[Node.SenderID] // Henter map med caborder for NODE vi snakker med atm

	for k := 0; k < _numFloors; k++ { // gjør endringer på map basert på map og melding fra node vi snakker med
		incomingCabstate := Node.OrderListCab[k]
		currentBackupState := CabBackup[k]
		switch {
		case (currentBackupState == Order_Inactive) && (incomingCabstate == Order_Pending):
			CabBackup[k] = Order_Pending

		case (currentBackupState == Order_Pending) && (incomingCabstate == Order_Pending || incomingCabstate == Order_Active):
			CabBackup[k] = Order_Active

		case (currentBackupState == Order_Active) && (incomingCabstate == Order_Inactive):
			CabBackup[k] = Order_Inactive

		default:
			continue
		}

	}
	e.CabBackupMap[Node.SenderID] = CabBackup // skriver ny status til map
}
