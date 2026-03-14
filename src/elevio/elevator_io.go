package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond
const numButtons = 3

var _initialized bool = false
var _numFloors int = 4
var NumFloors int = 4 //default verdi
var topFloor int = _numFloors - 1
var _mtx sync.Mutex
var _conn net.Conn

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_numFloors = numFloors
	NumFloors = numFloors //global variabel
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down MotorDirection = -1
	MD_Stop MotorDirection = 0
)

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

func SetMotorDirection(dir MotorDirection) {
	write([4]byte{1, byte(dir), 0, 0})
}

func SetButtonLamp(button ButtonType, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func SetFloorIndicator(floor int) {
	write([4]byte{3, byte(floor), 0, 0})
}

func SetDoorOpenLamp(value bool) {
	write([4]byte{4, toByte(value), 0, 0})
}

func SetStopLamp(value bool) {
	write([4]byte{5, toByte(value), 0, 0})
}

func PollButtons(receiver chan<- ButtonEvent) {
	prevButtonState := make([][3]bool, _numFloors)
	for {
		time.Sleep(_pollRate)
		for floor := 0; floor < _numFloors; floor++ {
			for button := ButtonType(0); button < 3; button++ {
				pressed := GetButton(button, floor) //Evt bytte til isPressed?
				if pressed != prevButtonState[floor][button] && pressed != false {
					receiver <- ButtonEvent{floor, ButtonType(button)}
				}
				prevButtonState[floor][button] = pressed
			}
		}
	}
}

func PollFloorSensor(receiver chan<- int, btnPress <-chan bool, hasActiveOrders func() bool) {
	prevFloorState := -1 //Eller lastFloor
	for {

		time.Sleep(_pollRate)
		currentFloor := GetFloor() //Er dette et greit navn?

		buttonPressed := false

		select {
		case <-btnPress:
			buttonPressed = true
		default:
			buttonPressed = false
		}

		if (currentFloor != prevFloorState && currentFloor != -1) || (currentFloor != -1 && buttonPressed) || (hasActiveOrders() && currentFloor != -1) { //denne fyrer hvert 20ms når aktive ordre, prøvd å fjerne men det ødela absolutt alt annet
			receiver <- currentFloor
		}
		prevFloorState = currentFloor
	}
}

func PollStopButton(receiver chan<- bool) {
	prevStopState := false
	for {
		time.Sleep(_pollRate)
		stopPressed := GetStop()
		if stopPressed != prevStopState {
			receiver <- stopPressed
		}
		prevStopState = stopPressed
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prevObstructionState := false
	for {
		time.Sleep(_pollRate)
		obstructionActiv := GetObstruction()
		if obstructionActiv != prevObstructionState {
			receiver <- obstructionActiv
		}
		prevObstructionState = obstructionActiv
	}
}

func GetButton(button ButtonType, floor int) bool {
	response := read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(response[1])
}

func GetFloor() int {
	response := read([4]byte{7, 0, 0, 0})
	if response[1] != 0 {
		return int(response[2])
	} else {
		return -1
	}
}

func GetStop() bool {
	response := read([4]byte{8, 0, 0, 0})
	return toBool(response[1])
}

func GetObstruction() bool {
	response := read([4]byte{9, 0, 0, 0})
	return toBool(response[1])
}

func read(in [4]byte) [4]byte {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = _conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func toByte(value bool) byte {
	var result byte = 0
	if value {
		result = 1
	}
	return result
}

func toBool(value byte) bool {
	var result bool = false
	if value != 0 {
		result = true
	}
	return result
}

//Tror dette skal fungere like bra, og er litt lettere å lese
//func toByte(value bool) byte {  
//	if value {   
//		return 1  
//	}  
//	return 0
//}  
//
//func toBool(value byte) bool {  
//	return value != 0 
//}