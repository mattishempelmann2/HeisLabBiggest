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
	const numFloors = 4

	otherNodesMap := make(map[string]elev.ElevatorMessage) //Map to store messages from other nodes
	lastSeenMap := make(map[string]time.Time)              //Map to note when node x last seen

	timeOutTicker := time.NewTicker(500 * time.Millisecond)
	nodeTimeout := 4 * time.Second

	doorTimeOpen := 3 * time.Second
	doorTimer := time.NewTimer(doorTimeOpen)
	doorTimer.Stop()

	obstructionLimit := 8 * time.Second
	doorObstructedTimer := time.NewTimer(obstructionLimit)
	doorObstructedTimer.Stop()

	lastFloorChangeTime := time.Now()
	motorWatchdog := time.NewTicker(1 * time.Second)

	sendTicker := time.NewTicker(10 * time.Millisecond) //Send interval

	localID := flag.Int("port", 15657, "UDP PORT")
	flag.Parse()

	networkStatusOut := make(chan elev.ElevatorMessage)
	networkStatusIn := make(chan elev.ElevatorMessage)

	go bcast.Transmitter(20013, networkStatusOut)
	go bcast.Receiver(20013, networkStatusIn)

	address := fmt.Sprintf("localhost:%d", *localID)
	elevio.Init(address, numFloors)

	elevator := &elev.Elevator{}
	elevator.CabInit(address, numFloors)

	buttonEventChan := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	obstructionEvent := make(chan bool)
	stopEvent := make(chan bool)
	buttonPressCh := make(chan bool, 1)

	go elevio.PollButtons(buttonEventChan)
	go elevio.PollFloorSensor(floorEvent, buttonPressCh, elevator.ActiveOrders)
	go elevio.PollObstructionSwitch(obstructionEvent)
	go elevio.PollStopButton(stopEvent)

	for {
		runCost := false //Flag to run cost function

		select {
		case buttonEvent := <-buttonEventChan:
			elevator.UpdateElevatorOrder(buttonEvent)
			buttonPressCh <- true
			elevator.GoingWrongway(&buttonEvent)
			runCost = true
		case newFloor := <-floorEvent:
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
				elevator.ExecuteOrder()
				if elevator.State.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(doorTimeOpen)
				}
			}
			runCost = true

		case <-doorTimer.C:
			elevator.DoorTimeHandler(doorTimer, doorTimeOpen)
			runCost = true

		case <-sendTicker.C:
			if elevator.State.Stuck {
				continue
			}
			elevator.SendStatus(address, networkStatusOut)
		case msg := <-networkStatusIn:
			if (msg.SenderID == address) || msg.MessageID <= otherNodesMap[msg.SenderID].MessageID || elevator.State.Stuck {
				continue
			}

			lastSeenMap[msg.SenderID] = time.Now()

			if !elevator.OtherNodes.Alive[msg.SenderID] {
				elevator.OtherNodes.Alive[msg.SenderID] = true
				fmt.Printf("Node %s connected \n", msg.SenderID)
				runCost = true
			}

			stateChanged := elevator.StateChanged(msg, otherNodesMap)

			otherNodesMap[msg.SenderID] = msg
			elevator.CabBackupFunc(msg)
			elevator.HallConsensus(msg, otherNodesMap)

			if stateChanged {
				runCost = true
			}
		case <-timeOutTicker.C:
			for id, lastTime := range lastSeenMap {
				if elevator.OtherNodes.Alive[id] && time.Since(lastTime) > nodeTimeout {
					elevator.OtherNodes.Alive[id] = false
					fmt.Printf("Watchdog: Node %s timed out! Marking as dead.\n", id)
					delete(otherNodesMap, id)
					runCost = true
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

		case obstruction := <-obstructionEvent:
			elevator.ObstructionHandler(obstruction, doorObstructedTimer, obstructionLimit, doorTimer, doorTimeOpen)

		case stopPressed := <-stopEvent:
			if stopPressed {
				elevio.SetStopLamp(true)
				elevator.CabInit(address, numFloors)
				elevio.SetStopLamp(false)
			}
		}
		if runCost {
			result := cost.CostFunc(cost.MakeHRAInput(*elevator, otherNodesMap))[address]
			if result != nil {
				elevator.Orders.Assigned = result
			}
			elevator.UpdateHallLights()
		}
	}
}
