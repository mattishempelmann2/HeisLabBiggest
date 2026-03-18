package elev

import (
	"fmt"
	"heis/src/elevio"
	"time"
)

func (e *Elevator) StoppFloor() {
	e.SetElevMotorDirection(0)
	e.State.DoorOpen = true
	e.SetElevDoorOpenLamp(true)
	e.ClearOrderFloor()

}

func (e *Elevator) ChooseDirection() elevio.MotorDirection {
	switch e.State.Direction {
	case elevio.MD_Up:
		if e.HasOrderAbove() {
			return elevio.MD_Up
		} else if e.HasOrderBelow() {
			return elevio.MD_Down
		}
		return elevio.MD_Stop
	case elevio.MD_Down:
		if e.HasOrderBelow() {
			return elevio.MD_Down
		} else if e.HasOrderAbove() {
			return elevio.MD_Up
		}
		return elevio.MD_Stop
	case elevio.MD_Stop:
		if e.State.PrevDirection == elevio.MD_Down {
			if e.HasOrderBelow() {
				return elevio.MD_Down
			} else if e.HasOrderAbove() {
				return elevio.MD_Up
			}
		} else {
			if e.HasOrderAbove() {
				return elevio.MD_Up
			} else if e.HasOrderBelow() {
				return elevio.MD_Down
			}
		}
		return elevio.MD_Stop
	default:
		return elevio.MD_Stop
	}
}

func (e *Elevator) ShouldStop() bool {
	if e.Orders.ListCab[e.State.Floor] == Order_Active {
		return true
	}
	dir := e.State.Direction
	if dir == elevio.MD_Stop {
		dir = e.State.PrevDirection
	}
	switch dir {
	case elevio.MD_Up:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp] || (!e.HasOrderAbove() && e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown])

	case elevio.MD_Down:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown] || (!e.HasOrderBelow() && e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp])

	default:
		return e.Orders.Assigned[e.State.Floor][elevio.BT_HallDown] || e.Orders.Assigned[e.State.Floor][elevio.BT_HallUp]
	}

}

func (e *Elevator) ExecuteOrder() {
	if e.ShouldStop() {
		e.StoppFloor()
		return
	}
	nextDir := e.ChooseDirection()
	if nextDir != e.State.Direction {
		fmt.Printf("Going %s \n", dirMap[int(nextDir)])
	}
	e.SetElevMotorDirection(nextDir)
}

func (e *Elevator) RunningAlone() bool {
	for id := range e.OtherNodes.Alive {
		if e.OtherNodes.Alive[id] {
			return false
		}
	}
	return true
}

func (e *Elevator) GoingWrongway(event *elevio.ButtonEvent) {
	if event.Button == elevio.BT_Cab && e.State.DoorOpen {
		e.State.AnnouncementPending = (e.State.AnnouncedDirection == elevio.MD_Up && event.Floor < e.State.Floor) || (e.State.AnnouncedDirection == elevio.MD_Down && event.Floor > e.State.Floor)
	}
}

func (e *Elevator) DoorTimeHandler(doorTimer *time.Timer, time time.Duration) {
	if e.State.Obstructed {
		fmt.Printf("Cab obstructed, keeping door open \n")
		doorTimer.Reset(time)
	} else if e.State.AnnouncementPending {
		e.State.AnnouncementPending = false
		e.State.AnnouncedDirection = elevio.MD_Stop
		fmt.Printf("Changing Directions \n")
		doorTimer.Reset(time)
	} else {
		fmt.Printf("Door closing \n")
		e.State.DoorOpen = false
		e.SetElevDoorOpenLamp(false)

		if e.State.Stuck {
			e.State.Stuck = false //Go back online after obstruction is resolved.
		}

		e.ExecuteOrder()
		if e.State.DoorOpen {
			doorTimer.Reset(time) //Handles edge case after reboot. The elevator would not start the doortimer if not.
		}
	}
}

func (e *Elevator) StateChanged(msg ElevatorMessage, otherNodes map[string]ElevatorMessage) bool {
	return (!HallOrdersEqual(msg.OrderListHall, otherNodes[msg.SenderID].OrderListHall)) || !CabOrdersEqual(msg.OrderListCab, otherNodes[msg.SenderID].OrderListCab)
}

func (e *Elevator) StuckHandler(lastFloorChangeTime *time.Time) {
	if e.State.Direction == elevio.MD_Stop {
		*lastFloorChangeTime = time.Now()
	}
	movingButStuck := (e.State.Direction != elevio.MD_Stop) && (time.Since(*lastFloorChangeTime) > 3500*time.Millisecond)
	if movingButStuck && !e.State.Stuck {
		fmt.Printf("Motor is stuck\n")
		e.State.Stuck = true
		e.SetElevMotorDirection(elevio.MD_Stop)
	}
	if e.State.Stuck && !e.State.DoorOpen {
		*lastFloorChangeTime = time.Now()
		e.ExecuteOrder()
	}
}

func (e *Elevator) ObstructionHandler(obstruction bool, doorObstructedTimer *time.Timer, obstructionLimit time.Duration, doorTimer *time.Timer, doorTimeOpen time.Duration) {
	e.State.Obstructed = obstruction
	fmt.Printf("Obstruction: %v \n", e.State.Obstructed)
	doorObstructedTimer.Reset(obstructionLimit)
	if !obstruction && e.State.DoorOpen {
		doorTimer.Reset(doorTimeOpen)
		doorObstructedTimer.Stop()
	}
}
