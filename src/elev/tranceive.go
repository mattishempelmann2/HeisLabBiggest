package elev

func (e *Elevator) SendStatus(address string, networkStatusOut chan<- ElevatorMessage) {
	cabBackUpCopy := make(map[string][]OrderStatus)

	for nodeID, cabOrders := range e.Orders.CabBackupList { //tar deep copy, denne kjører fullstendig i denne casen, gjør at programm ikke kræsjer ved sending samtidig som knappetrykk
		cabOrdersCopy := make([]OrderStatus, len(cabOrders)) //lager ny liste med same lengde
		copy(cabOrdersCopy, cabOrders)                       //kopierer over verdier
		cabBackUpCopy[nodeID] = cabOrdersCopy                //Fyller inn i mapet vi laget for å sende
	}

	msg := ElevatorMessage{ //Lager statusmelding
		SenderID:      address,
		CurrentFloor:  e.State.Floor,
		Direction:     int(e.State.Direction),
		OrderListHall: e.Orders.ListHall,
		OrderListCab:  e.Orders.ListCab,
		CabBackupMap:  cabBackUpCopy,
		MessageID:     e.OtherNodes.MessageCount,
		DoorOpen:      e.State.DoorOpen, //bruke counter som MsgID
		Behaviour:     e.State.Behaviour,
	}
	networkStatusOut <- msg //sende
	e.OtherNodes.MessageCount++
}
