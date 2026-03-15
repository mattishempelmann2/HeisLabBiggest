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
	otherNodes := make(map[string]elev.ElevatorStatus) //denne gis til costfunc ER ikke denne lokal? endra den til det ihvertfall
	lastSeen := make(map[string]time.Time)             //map for å notere når node_x sist sett

	watchdogTicker := time.NewTicker(500 * time.Millisecond) //sjekk 2 gang i sekund om node død
	nodeTimeout := 4 * time.Second                           // juster om vi må

	doorTimeOpen := 3 * time.Second
	doorTimer := time.NewTimer(doorTimeOpen) //må startes/resetes manuelt
	doorTimer.Stop()                         // Timer starter når definert, stoppe så den ikke fucker opp states

	obstructionLimit := 8 * time.Second
	doorObstructedTimer := time.NewTimer(obstructionLimit)
	doorObstructedTimer.Stop()

	lastFloorChangeTime := time.Now()
	motorWatchdog := time.NewTicker(1 * time.Second)

	sendTicker := time.NewTicker(10 * time.Millisecond) // ticker = går av periodically forever, hvor ofte sender vi status

	localID := flag.Int("port", 15657, "UDP PORT") // bruke noe
	flag.Parse()

	networkStatusOut := make(chan elev.ElevatorStatus) //channel med status, Lokale??
	networkStatusIn := make(chan elev.ElevatorStatus)  //Lokale??

	go bcast.Transmitter(20013, networkStatusOut) //idk hvilken port som er korrekt
	go bcast.Receiver(20013, networkStatusIn)

	const numFloors = 4
	address := fmt.Sprintf("localhost:%d", *localID) //slipper å manuelt skrive inn argument til init
	elevio.Init(address, numFloors)

	elevator := &elev.Elevator{}
	elevator.CabInit(address, numFloors) //initsialisere med adresse og antall etasjer

	buttonEvents := make(chan elevio.ButtonEvent)
	floorEvents := make(chan int)
	obstructionEvents := make(chan bool)
	stopEvents := make(chan bool)
	buttonPressCh := make(chan bool, 1) //buffered channel to prevent deadlocks Lokal??

	go elevio.PollButtons(buttonEvents)
	go elevio.PollFloorSensor(floorEvents, buttonPressCh, elevator.ActiveOrders)
	go elevio.PollObstructionSwitch(obstructionEvents)
	go elevio.PollStopButton(stopEvents)

	for {
		runCost := false

		select {
		case buttonEvent := <-buttonEvents: //knappetrykk
			elevator.UpdateElevatorOrder(buttonEvent)
			buttonPressCh <- true

			if buttonEvent.Button == elevio.BT_Cab && elevator.DoorOpen {
				goingWrongWay := (elevator.AnnouncedDirection == elevio.MD_Up && buttonEvent.Floor < elevator.Floor) || (elevator.AnnouncedDirection == elevio.MD_Down && buttonEvent.Floor > elevator.Floor)
				if goingWrongWay {
					elevator.AnnouncementPending = true
				}
			}

			runCost = true
		case newFloor := <-floorEvents: //etasjeupdate
			if newFloor != elevator.Floor {
				lastFloorChangeTime = time.Now()
				if elevator.Stuck {
					elevator.Stuck = false
					fmt.Printf("Motor drive recovered \n")
				}
			}
			elevio.SetFloorIndicator(newFloor)
			elevator.UpdateFloor(newFloor)

			if !elevator.DoorOpen {
				elevator.ExecuteOrder2() // denne åpner dør
				if elevator.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(doorTimeOpen)
				}
			}
			runCost = true

		case <-doorTimer.C: //timer etter dør åpen
			if elevator.Obstructed {
				fmt.Printf("Cab obstructed, keeping door open \n")
				doorTimer.Reset(doorTimeOpen)
			} else if elevator.AnnouncementPending {
				elevator.AnnouncementPending = false
				elevator.AnnouncedDirection = elevio.MD_Stop
				fmt.Printf("Changing Directions \n")
				doorTimer.Reset(doorTimeOpen)
			} else {
				fmt.Printf("Door closing \n")
				elevator.DoorOpen = false
				elevator.SetElevDoorOpenLamp(false)

				if elevator.Stuck {
					elevator.Stuck = false // Hvis ikke forblir vi stuck etter at vi har fjernet obstruction, da starter vi aldri å sende igjen
				}

				elevator.ExecuteOrder2()
				if elevator.DoorOpen {
					doorTimer.Reset(doorTimeOpen) // Viktig siden, hvis vi har cab order til 0.etasje etter reboot, blir vi stuck uten, da dører åpnes uten å resete timer
				}
			}
			runCost = true

		case <-sendTicker.C: //Periodisk statusupdate
			if elevator.Stuck {
				continue
			}

			cabBackUpCopy := make(map[string][]elev.OrderStatus)

			for nodeID, cabOrders := range elevator.CabBackupMap { //tar deep copy, denne kjører fullstendig i denne casen, gjør at programm ikke kræsjer ved sending samtidig som knappetrykk
				cabOrdersCopy := make([]elev.OrderStatus, len(cabOrders)) //lager ny liste med same lengde
				copy(cabOrdersCopy, cabOrders)                            //kopierer over verdier
				cabBackUpCopy[nodeID] = cabOrdersCopy                     //Fyller inn i mapet vi laget for å sende
			}

			msg := elev.ElevatorStatus{ //Lager statusmelding
				SenderID:      address,
				CurrentFloor:  elevator.Floor,
				Direction:     int(elevator.Direction),
				OrderListHall: elevator.OrderListHall,
				OrderListCab:  elevator.OrderListCab,
				CabBackupMap:  cabBackUpCopy,
				MsgID:         elevator.MsgCount,
				DoorOpen:      elevator.DoorOpen, //bruke counter som MsgID
				Behaviour:     elevator.Behaviour,
			}
			networkStatusOut <- msg //sende
			elevator.MsgCount++

		case msg := <-networkStatusIn: //Mottar status update
			if (msg.SenderID == address) || msg.MsgID <= otherNodes[msg.SenderID].MsgID || elevator.Stuck {
				continue
			}

			lastSeen[msg.SenderID] = time.Now()

			if !elevator.AliveNodes[msg.SenderID] {
				elevator.AliveNodes[msg.SenderID] = true // setter true dersom den ikke er det
				fmt.Printf("Node %s connected \n", msg.SenderID)
				runCost = true //beregn på nytt, har fått ny node i systemet
			}

			stateChanged := (!elev.HallOrdersEqual(msg.OrderListHall, otherNodes[msg.SenderID].OrderListHall)) || !elev.CabOrdersEqual(msg.OrderListCab, otherNodes[msg.SenderID].OrderListCab) // Sjekk om state changed, sparer print og beregning
			otherNodes[msg.SenderID] = msg
			elevator.CabBackupFunc(msg)              // back up cab orders fra melding mottat
			elevator.SteinSaksPapir(msg, otherNodes) // hvis ikke egen eller gammel melding, gjør steinsakspapir algebra                                                                                                                                     //ta vare på siste msg

			if stateChanged { // kun print/gjør beregning ved endring, slipper spam
				PrintOrderMatrix(msg)
				runCost = true
			}
		case <-watchdogTicker.C:
			for id, lastTime := range lastSeen {
				if elevator.AliveNodes[id] && time.Since(lastTime) > nodeTimeout {
					elevator.AliveNodes[id] = false // marker som død
					fmt.Printf("Watchdog: Node %s timed out! Marking as dead.\n", id)
					delete(otherNodes, id) // fjern fra otherNodes liste cost funk bruker
					runCost = true         // beregn på nytt
				}
			}
		case <-doorObstructedTimer.C:
			if elevator.Obstructed && elevator.DoorOpen {
				fmt.Printf("Door stuck due to obstruction \n")
				elevator.Stuck = true
				elevator.SetElevMotorDirection(elevio.MD_Stop)
			}

		case <-motorWatchdog.C:
			if elevator.Direction == elevio.MD_Stop {
				lastFloorChangeTime = time.Now()
			}

			movingButStuck := (elevator.Direction != elevio.MD_Stop) && (time.Since(lastFloorChangeTime) > 3500*time.Millisecond)

			if movingButStuck && !elevator.Stuck {
				fmt.Printf("Motor is stuck\n")
				elevator.Stuck = true
				elevator.SetElevMotorDirection(elevio.MD_Stop)
			}

			if elevator.Stuck && !elevator.DoorOpen {
				lastFloorChangeTime = time.Now()
				elevator.ExecuteOrder2()
			}

		case obstruction := <-obstructionEvents: //Obstruksjonsbryter
			elevator.Obstructed = obstruction
			fmt.Printf("Obstruction: %v \n", elevator.Obstructed)
			doorObstructedTimer.Reset(obstructionLimit)
			if !obstruction && elevator.DoorOpen {
				doorTimer.Reset(doorTimeOpen)
				doorObstructedTimer.Stop()
			}

		case stopPressed := <-stopEvents: //stop bryter
			if stopPressed {
				elevio.SetStopLamp(true)
				elevator.CabInit(address, numFloors)
				elevio.SetStopLamp(false) //Init func
			}
		}
		if runCost {
			result := cost.CostFunc(cost.MakeHRAInput(*elevator, otherNodes))[address]
			if result != nil {
				elevator.AssignedOrders = result
			}
			elevator.UpdateHallLights() // synkroniserer hall lights
		}
	}
}

func PrintOrderMatrix(e elev.ElevatorStatus) {
	fmt.Printf("   %s  %s  %s\n", "Up", "Dn", "Cab") // Header (Optional)
	for floor := 0; floor < 4; floor++ {
		fmt.Printf("F%d ", floor)
		for button := 0; button < 2; button++ {
			switch {
			case e.OrderListHall[floor][button] == elev.Order_Active:
				fmt.Printf("[%s] ", "X")
			case e.OrderListHall[floor][button] == elev.Order_Pending:
				fmt.Printf("[%s] ", "P")
			case e.OrderListHall[floor][button] == elev.Order_PendingInactive:
				fmt.Printf("[%s] ", "C")
			default:
				fmt.Printf("[%s] ", " ")
			}
		}
		switch {
		case e.OrderListCab[floor] == elev.Order_Active:
			fmt.Printf("[%s] ", "X")
		case e.OrderListCab[floor] == elev.Order_Pending:
			fmt.Printf("[%s] ", "P")
		default:
			fmt.Printf("[%s] ", " ")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("msgID: %d, from NodeID: %s \n", e.MsgID, e.SenderID)
}
