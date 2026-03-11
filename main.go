package main

import (
	"flag"
	"fmt"
	cost "heis/src/cost_func"
	"heis/src/elev"
	"heis/src/elevio"
	"heis/src/network/bcast"
	"time"
)

func main() {
	OtherNodes := make(map[string]elev.ElevatorStatus) //denne gis til costfunc
	lastSeen := make(map[string]time.Time)             //map for å notere når node_x sist sett

	watchdogTicker := time.NewTicker(500 * time.Millisecond) //sjekk 2 gang i sekund om node død
	nodeTimeout := 3 * time.Second                           // juster om vi må

	doorTimeOpen := 3 * time.Second
	doorTimer := time.NewTimer(doorTimeOpen) //må startes/resetes manuelt
	doorTimer.Stop()                         // Timer starter når definert, stoppe så den ikke fucker opp states

	lastFloorChangeTime := time.Now()
	motorWatchdog := time.NewTicker(1 * time.Second)

	sendTicker := time.NewTicker(10 * time.Millisecond) // ticker = går av periodically forever, hvor ofte sender vi status

	localID := flag.Int("port", 15657, "UDP PORT") // bruke noe
	flag.Parse()

	StatusTx := make(chan elev.ElevatorStatus) //channel med status
	StatusRx := make(chan elev.ElevatorStatus)

	go bcast.Transmitter(20013, StatusTx) //idk hvilken port som er korrekt
	go bcast.Receiver(20013, StatusRx)

	const NumFloors = 4
	address := fmt.Sprintf("localhost:%d", *localID) //slipper å manuelt skrive inn argument til init
	elevio.Init(address, NumFloors)

	cab1 := &elev.Elevator{}
	cab1.CabInit(address, NumFloors) //Init func

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	BtnPress := make(chan bool, 1) //buffered channel to prevent deadlocks

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors, BtnPress, cab1.ActiveOrders)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	for {
		runCost := false

		select {
		case a := <-drv_buttons: //knappetrykk
			cab1.UpdateElevatorOrder(a)
			BtnPress <- true

			if a.Button == elevio.BT_Cab && cab1.DoorOpen {
				goingWrongWay := (cab1.AnnouncedDirection == elevio.MD_Up && a.Floor < cab1.Floor) || (cab1.AnnouncedDirection == elevio.MD_Down && a.Floor > cab1.Floor)
				if goingWrongWay {
					cab1.AnnouncementPending = true
				}
			}

			runCost = true
		case a := <-drv_floors: //etasjeupdate
			elevio.SetFloorIndicator(a)
			cab1.UpdateFloor(a)
			lastFloorChangeTime = time.Now()
			if cab1.Stuck {
				cab1.Stuck = false
				fmt.Printf("Motor drive recovered \n")
			}

			if !cab1.DoorOpen {
				cab1.ExecuteOrder2() // denne åpner dør
				if cab1.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(doorTimeOpen)
				}
			}
			runCost = true

		case <-doorTimer.C: //timer etter dør åpen
			if cab1.Obstructed {
				fmt.Printf("Cab obstructed, keeping door open \n")
				doorTimer.Reset(doorTimeOpen)
			} else if cab1.AnnouncementPending {
				cab1.AnnouncementPending = false
				cab1.AnnouncedDirection = elevio.MD_Stop
				fmt.Printf("Changing Directions \n")
				doorTimer.Reset(doorTimeOpen)
			} else {
				fmt.Printf("Door closing \n")
				cab1.DoorOpen = false
				cab1.SetElevDoorOpenLamp(false)

				if cab1.Stuck {
					cab1.Stuck = false // Hvis ikke forblir vi stuck etter at vi har fjernet obstruction, da starter vi aldri å sende igjen
				}

				cab1.ExecuteOrder2()
				if cab1.DoorOpen {
					doorTimer.Reset(doorTimeOpen) // Viktig siden, hvis vi har cab order til 0.etasje etter reboot, blir vi stuck uten, da dører åpnes uten å resete timer
				}
			}
			runCost = true

		case <-sendTicker.C: //Periodisk statusupdate
			if cab1.Stuck {
				continue
			}

			cabBackUpCopy := make(map[string][]elev.OrderStatus)

			for nodeID, cabOrders := range cab1.CabBackupMap { //tar deep copy, denne kjører fullstendig i denne casen, gjør at programm ikke kræsjer ved sending samtidig som knappetrykk
				cabOrdersCopy := make([]elev.OrderStatus, len(cabOrders)) //lager ny liste med same lengde
				copy(cabOrdersCopy, cabOrders)                            //kopierer over verdier
				cabBackUpCopy[nodeID] = cabOrdersCopy                     //Fyller inn i mapet vi laget for å sende
			}

			msg := elev.ElevatorStatus{ //Lager statusmelding
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
			if (msg.SenderID == address) || msg.MsgID <= OtherNodes[msg.SenderID].MsgID {
				continue
			}

			lastSeen[msg.SenderID] = time.Now()

			if !cab1.AliveNodes[msg.SenderID] {
				cab1.AliveNodes[msg.SenderID] = true // setter true dersom den ikke er det
				fmt.Printf("Node %s connected \n", msg.SenderID)
				runCost = true //beregn på nytt, har fått ny node i systemet
			}

			stateChanged := (!elev.HallOrdersEqual(msg.OrderListHall, OtherNodes[msg.SenderID].OrderListHall)) || !elev.CabOrdersEqual(msg.OrderListCab, OtherNodes[msg.SenderID].OrderListCab) // Sjekk om state changed, sparer print og beregning
			OtherNodes[msg.SenderID] = msg
			cab1.CabBackupFunc(msg)              // back up cab orders fra melding mottat
			cab1.SteinSaksPapir(msg, OtherNodes) // hvis ikke egen eller gammel melding, gjør steinsakspapir algebra                                                                                                                                     //ta vare på siste msg

			if stateChanged { // kun print/gjør beregning ved endring, slipper spam
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
		case <-motorWatchdog.C:
			if cab1.Direction == elevio.MD_Stop && !cab1.DoorOpen {
				lastFloorChangeTime = time.Now()
			}

			movingButStuck := (cab1.Direction != elevio.MD_Stop) && (time.Since(lastFloorChangeTime) > 5*time.Second)
			doorStuck := cab1.Obstructed && cab1.DoorOpen && (time.Since(lastFloorChangeTime) > 10*time.Second)

			if (movingButStuck || doorStuck) && !cab1.Stuck {
				fmt.Printf("Cab is stuck (motor: %v, door: %v) \n", movingButStuck, doorStuck)
				cab1.Stuck = true
				cab1.SetElevMotorDirection(elevio.MD_Stop)
			}

			if cab1.Stuck && !cab1.DoorOpen {
				lastFloorChangeTime = time.Now()
				cab1.ExecuteOrder2()
			}

		case obstruction := <-drv_obstr: //Obstruksjonsbryter
			cab1.Obstructed = obstruction
			fmt.Printf("Obstruction: %v \n", cab1.Obstructed)
			if !obstruction && cab1.DoorOpen {
				doorTimer.Reset(3 * time.Second)
			}

		case a := <-drv_stop: //stop bryter
			fmt.Printf("%+v\n", a)
			for f := 0; f < NumFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					cab1.SetElevButtonLamp(b, f, false)
				}
			}
		}
		if runCost {
			result := cost.CostFunc(cost.MakeHRAInput(*cab1, OtherNodes))[address]
			if result != nil {
				cab1.AssignedOrders = result
			}
			cab1.UpdateHallLights() // synkroniserer hall lights
		}
	}
}

func PrintOrderMatrix(e elev.ElevatorStatus) {
	fmt.Printf("   %s  %s  %s\n", "Up", "Dn", "Cab") // Header (Optional)
	for f := 0; f < 4; f++ {
		fmt.Printf("F%d ", f)
		for b := 0; b < 2; b++ {
			switch {
			case e.OrderListHall[f][b] == elev.Order_Active:
				fmt.Printf("[%s] ", "X")
			case e.OrderListHall[f][b] == elev.Order_Pending:
				fmt.Printf("[%s] ", "P")
			case e.OrderListHall[f][b] == elev.Order_PendingInactive:
				fmt.Printf("[%s] ", "C")
			default:
				fmt.Printf("[%s] ", " ")
			}
		}
		switch {
		case e.OrderListCab[f] == elev.Order_Active:
			fmt.Printf("[%s] ", "X")
		case e.OrderListCab[f] == elev.Order_Pending:
			fmt.Printf("[%s] ", "P")
		default:
			fmt.Printf("[%s] ", " ")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("msgID: %d, from NodeID: %s \n", e.MsgID, e.SenderID)
}
