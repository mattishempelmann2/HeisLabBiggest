package main

import (
	"fmt"
	"heis/src/elevio"
	"heis/src/network/bcast"
	"time"
)

func PrintOrderMatrix(e elevio.ElevatorStatus) {
	fmt.Printf("   %s  %s  %s\n", "Up", "Dn", "Cab") // Header (Optional)
	for f := 0; f < 4; f++ {
		fmt.Printf("F%d ", f)
		for b := 0; b < 3; b++ {
			switch {
			case e.OrderList[f][b] == elevio.Order_Active:
				fmt.Printf("[%s] ", "X")
			case e.OrderList[f][b] == elevio.Order_Pending:
				fmt.Printf("[%s] ", "P")
			default:
				fmt.Printf("[%s] ", " ")
			}
		}
		fmt.Printf("\n")
	}
}

func main() {
	counter := 0 //
	lastSeenID := 0

	localID := 15657 // bruke noe

	StatusTx := make(chan elevio.ElevatorStatus) //channel med status
	StatusRx := make(chan elevio.ElevatorStatus)

	go bcast.Transmitter(20014, StatusTx) //idk hvilken port som er korrekt
	go bcast.Receiver(20014, StatusRx)

	sendTicker := time.NewTicker(500 * time.Millisecond) // ticker = går av periodically forever, hvor ofte sender vi status

	const numFloors = 4
	address := fmt.Sprintf("localhost:%d", localID)
	elevio.Init(address, numFloors)

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
	doorTimer.Stop()                            // Timer starter når definert, stoppe så den ikke fucker opp states

	for {
		select {
		case a := <-drv_buttons: //knappetrykk
			//fmt.Printf("%+v\n", a)
			//cab1.SetButtonLamp(a.Button, a.Floor, true)  //må gjøres noe med lys settes nå på i SteinSaksPapir
			cab1.UpdateElevatorOrder(a)
			BtnPress <- true

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

		case <-doorTimer.C: //timer etter dør åpen
			fmt.Printf("Door closing \n")
			cab1.DoorOpen = false
			cab1.SetDoorOpenLamp(false)
			cab1.ExecuteOrder()

		case <-sendTicker.C: //Periodisk statusupdate
			msg := elevio.ElevatorStatus{ //Lager statusmelding
				SenderID:     localID,
				CurrentFloor: cab1.Floor,
				Direction:    int(cab1.Retning),
				OrderList:    cab1.OrderList,
				MsgID:        counter, //bruke counter som MsgID
			}
			StatusTx <- msg //sende
			counter++       // dårlig quickfix, gjør om til medlemsvariabel senere

		case msg := <-StatusRx: //Mottar status update
			if (msg.SenderID == localID) || msg.MsgID < lastSeenID { //sjekk om dette er gammel melding
				continue
			}
			cab1.SteinSaksPapir(msg) // hvis ikke egen eller gammel melding, gjør steinsakspapir algebra

			lastSeenID = msg.MsgID // oppdater sist sett.
			//fmt.Printf("Received message from %d at floor %d \n", msg.SenderID, msg.CurrentFloor)
			PrintOrderMatrix(msg)

		case a := <-drv_obstr: //Obstruksjonsbryter
			fmt.Printf("%+v\n", a)
			if a {
				cab1.SetMotorDirection(elevio.MD_Stop)
			} else {
				cab1.SetMotorDirection(d)
			}

		case a := <-drv_stop: //stop bryter
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					cab1.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
