package main

import (
	"fmt"
	"heis/src/elevio"
	"heis/src/network/bcast"
	"time"
)

func main() {

	type ElevatorStatus struct { //det som sendes, health checks
		SenderID     string
		CurrentFloor int
		Direction    int
		OrderList    [4][3]bool
	}

	localID := "15657" // bruke noe

	StatusTx := make(chan ElevatorStatus) //channel med status
	StatusRx := make(chan ElevatorStatus)

	go bcast.Transmitter(20014, StatusTx) //idk hvilken port som er korrekt
	go bcast.Receiver(20014, StatusRx)

	sendTicker := time.NewTicker(1 * time.Second) // ticker = går av periodically forever

	numFloors := 4

	elevio.Init("localhost:15657", numFloors)

	cab1 := &elevio.Elevator{}
	cab1.CabInit() //Init func

	var d elevio.MotorDirection = elevio.MD_Up
	//cab1.SetMotorDirection(d)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	//OrderChan := make(chan elevio.ButtonEvent)
	BtnPress := make(chan bool)
	//Timerdone := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go cab1.PollFloorSensor(drv_floors, BtnPress)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	doorTimer := time.NewTimer(3 * time.Second) //må startes/resetes manuelt
	doorTimer.Stop()

	for {
		select {
		case a := <-drv_buttons:
			//fmt.Printf("%+v\n", a)
			cab1.SetButtonLamp(a.Button, a.Floor, true)
			cab1.UpdateOrderList(a)
			//OrderChan <- a
			BtnPress <- true

		case a := <-drv_floors:
			cab1.SetFloorIndicator(a)
			cab1.UpdateFloor(a)
			if !cab1.DoorOpen {
				cab1.ExecuteOrder()

				if cab1.DoorOpen {
					fmt.Printf("Door opening \n")
					doorTimer.Reset(3 * time.Second)
				}
			}

		case <-doorTimer.C:
			fmt.Printf("Door closing \n")
			cab1.DoorOpen = false
			cab1.SetDoorOpenLamp(false)
			cab1.ExecuteOrder()

		case <-sendTicker.C:
			msg := ElevatorStatus{
				SenderID:     localID,
				CurrentFloor: cab1.Floor,
				Direction:    int(cab1.Retning),
				OrderList:    cab1.OrderList,
			}
			StatusTx <- msg

		case msg := <-StatusRx:
			if msg.SenderID == localID {
				continue
			}
			fmt.Printf("Received message from %s at floor %d \n", msg.SenderID, msg.CurrentFloor)

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a)
			if a {
				cab1.SetMotorDirection(elevio.MD_Stop)
			} else {
				cab1.SetMotorDirection(d)
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					cab1.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
