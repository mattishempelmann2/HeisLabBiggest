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
	otherNodes := make(map[string]elev.ElevatorMessage) //denne gis til costfunc ER ikke denne lokal? endra den til det ihvertfall
	lastSeen := make(map[string]time.Time)              //map for å notere når node_x sist sett

	watchdogTicker := time.NewTicker(500 * time.Millisecond) //sjekk 2 gang i sekund om node død
	nodeTimeout := 4 * time.Second                           // juster om vi må

	doorTimeOpen := 3 * time.Second
	doorTimer := time.NewTimer(doorTimeOpen) //må startes/resetes manuelt
	doorTimer.Stop()                         // Timer starter når definert, stoppe så den ikke fucker opp State

	obstructionLimit := 8 * time.Second
	doorObstructedTimer := time.NewTimer(obstructionLimit)
	doorObstructedTimer.Stop()

	lastFloorChangeTime := time.Now()
	motorWatchdog := time.NewTicker(1 * time.Second)

	sendTicker := time.NewTicker(10 * time.Millisecond) // ticker = går av periodically forever, hvor ofte sender vi status

	localID := flag.Int("port", 15657, "UDP PORT") // bruke noe
	flag.Parse()

	networkStatusOut := make(chan elev.ElevatorMessage) //channel med status, Lokale??
	networkStatusIn := make(chan elev.ElevatorMessage)  //Lokale??

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
			elevator.GoingWrongway(&buttonEvent)

			runCost = true
		case newFloor := <-floorEvents: //etasjeupdate
			if newFloor != elevator.State.Floor {
				lastFloorChangeTime = time.Now()
				if elevator.State.Stuck {
					elevator.State.Stuck = false
					fmt.Printf("Motor drive recovered \n")
				}
			}
			elevio.SetFloorIndicator(newFloor)
			elevator.UpdateFloor(newFloor)

			if !elevator.State.DoorOpen {
				elevator.ExecuteOrder2() // denne åpner dør
				if elevator.State.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(doorTimeOpen)
				}
			}
			runCost = true

		case <-doorTimer.C: //timer etter dør åpen
			elevator.DoorTimeHandler(doorTimer, doorTimeOpen)
			runCost = true

		case <-sendTicker.C: //Periodisk statusupdate
			if elevator.State.Stuck {
				continue
			}
			elevator.SendStatus(address, networkStatusOut)
		case msg := <-networkStatusIn: //Mottar status update
			if (msg.SenderID == address) || msg.MessageID <= otherNodes[msg.SenderID].MessageID || elevator.State.Stuck {
				continue
			}

			lastSeen[msg.SenderID] = time.Now()

			if !elevator.OtherNodes.Alive[msg.SenderID] {
				elevator.OtherNodes.Alive[msg.SenderID] = true // setter true dersom den ikke er det
				fmt.Printf("Node %s connected \n", msg.SenderID)
				runCost = true //beregn på nytt, har fått ny node i systemet
			}

			stateChanged := elevator.StateChanged(msg, otherNodes)

			otherNodes[msg.SenderID] = msg
			elevator.CabBackupFunc(msg)              // back up cab orders fra melding mottat
			elevator.SteinSaksPapir(msg, otherNodes) // hvis ikke egen eller gammel melding, gjør steinsakspapir algebra                                                                                                                                     //ta vare på siste msg

			if stateChanged { // kun print/gjør beregning ved endring, slipper spam
				PrintOrderMatrix(msg)
				runCost = true
			}
		case <-watchdogTicker.C:
			for id, lastTime := range lastSeen {
				if elevator.OtherNodes.Alive[id] && time.Since(lastTime) > nodeTimeout {
					elevator.OtherNodes.Alive[id] = false // marker som død
					fmt.Printf("Watchdog: Node %s timed out! Marking as dead.\n", id)
					delete(otherNodes, id) // fjern fra otherNodes liste cost funk bruker
					runCost = true         // beregn på nytt
				}
			}
		case <-doorObstructedTimer.C:
			if elevator.State.Obstructed && elevator.State.DoorOpen {
				fmt.Printf("Door stuck due to obstruction \n")
				elevator.State.Stuck = true
				elevator.SetElevMotorDirection(elevio.MD_Stop)
			}
		case <-motorWatchdog.C:
			elevator.StuckHandler(&lastFloorChangeTime)

		case obstruction := <-obstructionEvents: //Obstruksjonsbryter
			elevator.ObstructionHandler(obstruction, doorObstructedTimer, obstructionLimit, doorTimer, doorTimeOpen)

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
				elevator.Orders.Assigned = result
			}
			elevator.UpdateHallLights() // synkroniserer hall lights
		}
	}
}

func PrintOrderMatrix(e elev.ElevatorMessage) {
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
	fmt.Printf("msgID: %d, from NodeID: %s \n", e.MessageID, e.SenderID)
}
