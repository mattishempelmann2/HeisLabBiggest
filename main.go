package main

import (
	"fmt"
	cost "heis/src/cost_func"
	"heis/src/elevio"
	"heis/src/network/bcast"
	"time"
)

func PrintOrderMatrix(e elevio.ElevatorStatus) {
	fmt.Printf("   %s  %s  %s\n", "Up", "Dn", "Cab") // Header (Optional)
	for f := 0; f < 4; f++ {
		fmt.Printf("F%d ", f)
		for b := 0; b < 2; b++ {
			switch {
			case e.OrderListHall[f][b] == elevio.Order_Active:
				fmt.Printf("[%s] ", "X")
			case e.OrderListHall[f][b] == elevio.Order_Pending:
				fmt.Printf("[%s] ", "P")
			default:
				fmt.Printf("[%s] ", " ")
			}
		}
		switch {
		case e.OrderListCab[f] == elevio.Order_Active:
			fmt.Printf("[%s] ", "X")
		case e.OrderListCab[f] == elevio.Order_Pending:
			fmt.Printf("[%s] ", "P")
		default:
			fmt.Printf("[%s] ", " ")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("msgID: %d, from NodeID: %s \n", e.MsgID, e.SenderID)
}

func main() {
	OtherNodes := make(map[string]elevio.ElevatorStatus)
	lastSeen := make(map[string]time.Time) //map for å notere når node_x sist sett
	watchdogTicker := time.NewTicker(500 * time.Millisecond)
	nodeTimeout := 3 * time.Second

	localID := 15657 // bruke noe

	StatusTx := make(chan elevio.ElevatorStatus) //channel med status
	StatusRx := make(chan elevio.ElevatorStatus)

	go bcast.Transmitter(20013, StatusTx) //idk hvilken port som er korrekt
	go bcast.Receiver(20013, StatusRx)

	sendTicker := time.NewTicker(10 * time.Millisecond) // ticker = går av periodically forever, hvor ofte sender vi status

	const NumFloors = 4
	address := fmt.Sprintf("localhost:%d", localID) //slipper å manuelt skrive inn argument til init
	elevio.Init(address, NumFloors)

	cab1 := &elevio.Elevator{}
	cab1.CabInit(address) //Init func

	var d elevio.MotorDirection = elevio.MD_Up // fjern etter hvert

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	BtnPress := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go cab1.PollFloorSensor(drv_floors, BtnPress)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	doorTimer := time.NewTimer(3 * time.Second) //må startes/resetes manuelt
	doorTimer.Stop()                            // Timer starter når definert, stoppe så den ikke fucker opp states

	for {
		runCost := false

		select {
		case a := <-drv_buttons: //knappetrykk
			cab1.UpdateElevatorOrder(a)
			BtnPress <- true
			runCost = true
		case a := <-drv_floors: //etasjeupdate
			cab1.SetFloorIndicator(a)
			cab1.UpdateFloor(a)
			if !cab1.DoorOpen {
				cab1.ExecuteOrder() // denne åpner dør

				if cab1.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(3 * time.Second)
				}
			}
			runCost = true

		case <-doorTimer.C: //timer etter dør åpen
			fmt.Printf("Door closing \n")
			cab1.DoorOpen = false
			cab1.SetDoorOpenLamp(false)
			cab1.ExecuteOrder()
			runCost = true

		case <-sendTicker.C: //Periodisk statusupdate
			cabBackUpCopy := make(map[string][4]elevio.OrderStatus)

			for key, value := range cab1.CabBackupMap {
				cabBackUpCopy[key] = value
			}

			msg := elevio.ElevatorStatus{ //Lager statusmelding
				SenderID:      address,
				CurrentFloor:  cab1.Floor,
				Direction:     int(cab1.Direction),
				OrderListHall: cab1.OrderListHall,
				OrderListCab:  cab1.OrderListCab,
				CabBackupMap:  cabBackUpCopy,
				MsgID:         cab1.MsgCount,
				DoorOpen:      cab1.DoorOpen, //bruke counter som MsgID
				Behaviour:     cab1.Behaviour,
			}
			StatusTx <- msg //sende
			cab1.MsgCount++

		case msg := <-StatusRx: //Mottar status update
			if (msg.SenderID == address) || msg.MsgID < OtherNodes[msg.SenderID].MsgID {
				continue
			}

			lastSeen[msg.SenderID] = time.Now()

			if !cab1.AliveNodes[msg.SenderID] {
				cab1.AliveNodes[msg.SenderID] = true // setter true dersom den ikke er det
				fmt.Printf("Node %s connected", msg.SenderID)
				runCost = true //beregn på nytt, har fått ny node i systemet
			}

			cab1.CabBackupFunc(msg)  // back up cab orders fra melding mottat
			cab1.SteinSaksPapir(msg) // hvis ikke egen eller gammel melding, gjør steinsakspapir algebra

			stateChanged := (msg.OrderListHall != OtherNodes[msg.SenderID].OrderListHall) || (msg.OrderListCab != OtherNodes[msg.SenderID].OrderListCab) // Sjekk om state changed, sparer print og beregning
			OtherNodes[msg.SenderID] = msg

			if stateChanged { // kun print ved endring, slipper spam
				PrintOrderMatrix(msg)
				runCost = true
			}
		case <-watchdogTicker.C:
			for id, lastTime := range lastSeen {
				if cab1.AliveNodes[id] && time.Since(lastTime) > nodeTimeout {
					cab1.AliveNodes[id] = false // marker som død
					fmt.Printf("Watchdog: Node %s timed out! Marking as dead.\n", id)
					delete(OtherNodes, id) // fjern fra othernodes liste cost funk bruker
					runCost = true         // beregn på nytt
				}
			}
		case a := <-drv_obstr: //Obstruksjonsbryter
			fmt.Printf("%+v\n", a)
			if a {
				cab1.SetMotorDirection(elevio.MD_Stop)
			} else {
				cab1.SetMotorDirection(d)
			}

		case a := <-drv_stop: //stop bryter
			fmt.Printf("%+v\n", a)
			for f := 0; f < NumFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					cab1.SetButtonLamp(b, f, false)
				}
			}
		}
		if runCost {
			cab1.AssignedOrders = cost.CostFunc(cost.MakeHRAInput(*cab1, OtherNodes))[address] // mp fikse slik at den ikke bruker døde nodes i beregning
			cab1.UpdateHallLights()
		}
	}
}
