package elev

func (e *Elevator) SendStatus(address string, networkStatusOut chan<- ElevatorMessage) {
	cabBackUpCopy := make(map[string][]OrderStatus)

	for nodeID, cabOrders := range e.Orders.CabBackupList {
		cabOrdersCopy := make([]OrderStatus, len(cabOrders))
		copy(cabOrdersCopy, cabOrders)
		cabBackUpCopy[nodeID] = cabOrdersCopy
	}

	message := ElevatorMessage{
		SenderID:      address,
		CurrentFloor:  e.State.Floor,
		Direction:     int(e.State.Direction),
		OrderListHall: e.Orders.ListHall,
		OrderListCab:  e.Orders.ListCab,
		CabBackupMap:  cabBackUpCopy,
		MessageID:     e.OtherNodes.MessageCount,
		DoorOpen:      e.State.DoorOpen,
		Behaviour:     e.State.Behaviour,
	}
	networkStatusOut <- message
	e.OtherNodes.MessageCount++
}
