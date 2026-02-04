package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

var _initialized bool = false
var _numFloors int = 4
var topFloor int = _numFloors - 1
var _mtx sync.Mutex
var _conn net.Conn

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down                = -1
	MD_Stop                = 0
)

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown            = 1
	BT_Cab                 = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type Elevator struct {
	OrderList   [4][3]bool
	Floor       int
	Retning     MotorDirection
	PrevRetning MotorDirection
	DoorOpen    bool
}

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_numFloors = numFloors
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

func (e *Elevator) SetMotorDirection(dir MotorDirection) {
	write([4]byte{1, byte(dir), 0, 0})
	e.UpdateRetning(dir)
}

func (e *Elevator) SetButtonLamp(button ButtonType, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func (e *Elevator) SetFloorIndicator(floor int) {
	write([4]byte{3, byte(floor), 0, 0})
}

func (e *Elevator) SetDoorOpenLamp(value bool) {
	write([4]byte{4, toByte(value), 0, 0})
}

func (e *Elevator) SetStopLamp(value bool) {
	write([4]byte{5, toByte(value), 0, 0})
}

func (e *Elevator) UpdateOrderList(Order ButtonEvent) {

	e.UpdateElevatorOrder(Order)
	//for req := range Orders {
	//	e.UpdateElevatorOrder(req)
	//	fmt.Printf("executing order: floor: %d button : %d", req.Floor, req.Button)
	//orderexecution
	//	}
}

func (e *Elevator) UpdateElevatorOrder(btn ButtonEvent) {
	e.OrderList[btn.Floor][btn.Button] = true
}

func (e *Elevator) UpdateFloor(Floor int) {
	if Floor != -1 {
		e.Floor = Floor
	}
}

func (e *Elevator) UpdateRetning(Retning MotorDirection) {
	e.PrevRetning = e.Retning
	e.Retning = Retning
}

func (e *Elevator) HasOrderAbove() bool {
	for i := e.Floor + 1; i < _numFloors; i++ {
		for j := 0; j < 3; j++ {
			if e.OrderList[i][j] == true {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) HasOrderBelow() bool {
	for i := e.Floor - 1; i >= 0; i-- {
		for j := 0; j < 3; j++ {
			if e.OrderList[i][j] == true {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) FloorOrder() bool {
	for i := 0; i < 3; i++ {
		if e.OrderList[e.Floor][i] == true {
			return true
		}
	}
	return false
}

func (e *Elevator) ActiveOrders() bool {
	for i := 0; i < topFloor; i++ {
		for j := 0; j < 3; j++ {
			if e.OrderList[i][j] == true {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) ClearOrderFloor() {
	for i := 0; i < topFloor; i++ {
		e.OrderList[e.Floor][i] = false
		e.SetButtonLamp(ButtonType(i), e.Floor, false)

	}
}

func (e *Elevator) DriveTo(floor int) { // fjern
	for e.Floor != floor {
		switch {
		case floor > e.Floor:
			e.SetMotorDirection(1)

		}
	}
}

func (e *Elevator) DoorTimer(SendDone chan<- bool) {
	time.Sleep(3 * time.Second)
	SendDone <- true
}

func (e *Elevator) StoppFloor() {
	e.SetMotorDirection(0)
	e.DoorOpen = true
	e.SetDoorOpenLamp(true)
	//time.Sleep(3 * time.Second)
	//e.DoorOpen = false
	//e.SetDoorOpenLamp(false)
	e.ClearOrderFloor()
}

func (e *Elevator) ExecuteOrder() {
	switch {
	case e.FloorOrder():
		switch {
		case e.OrderList[e.Floor][2] == true: // knapp cab
			e.StoppFloor()
		case (e.Retning == 1) && (e.OrderList[e.Floor][0] == true): //på tur oppover og knapp hall opp
			e.StoppFloor()
		case e.Retning == -1 && (e.OrderList[e.Floor][1] == true): // tur nedover knapp hall ned
			e.StoppFloor()
		case e.Retning == 0 && ((e.OrderList[e.Floor][1] == true) || (e.OrderList[e.Floor][0] == true)): // tur nedover knapp hall ned
			e.StoppFloor()
		default:
			break // mulig redundant
		}

	case e.HasOrderAbove():
		switch {
		case e.Retning == 1: // Har odre over er på tur opp -> fortsett opp
			e.SetMotorDirection(1)

		case e.Retning == -1: // HAr odre oveer er på tur ned -> fortsett ned
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 1) && e.Floor != topFloor: // Har ordre over, stoppet i etasje, var på tur opp og er ikke i toppetasjen -> kjør opp
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.PrevRetning == -1) && e.Floor != 0: // Har odre over, var på tur ned stopped i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 0) && e.Floor != topFloor: // Har odre over, var på tur ned stopped i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(1)
		
		case (e.Retning == 0) && (e.Floor == 0):
			e.SetMotorDirection(1)

		default:
			break
		}

	case e.HasOrderBelow():
		switch {
		case e.Retning == 1: // Har odre under er på tur opp -> fortsett opp
			e.SetMotorDirection(1)

		case e.Retning == -1: // HAr odre oveer er på tur ned -> fortsett ned
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 1) && e.Floor != topFloor: // Har ordre over, stoppet i etasje, var på tur opp og er ikke i toppetasjen -> kjør opp
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.PrevRetning == -1) && e.Floor != 0: // Har odre over, var på tur ned stopped i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 0) && e.Floor != 0: // Har odre over, var på tur ned stopped i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(-1)
		
		case (e.Retning == 0) && (e.Floor == topFloor):
			e.SetMotorDirection(-1)

		default:
			break
		}

	default:
		e.SetMotorDirection(0)
	}

}

func PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, _numFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < _numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := GetButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func (e *Elevator) PollFloorSensor(receiver chan<- int, btnPress <-chan bool) {
	prev := -1
	for {

		time.Sleep(_pollRate)
		v := GetFloor()

		buttonPressed := false

		select {
		case <-btnPress:
			buttonPressed = true
		default:
			buttonPressed = false
		}

		if (v != prev && v != -1) || (v != -1 && buttonPressed) || (e.ActiveOrders() && v != -1) {
			receiver <- v
		}
		prev = v
	}
}

func PollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func GetButton(button ButtonType, floor int) bool {
	a := read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

func GetFloor() int {
	a := read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func GetStop() bool {
	a := read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func GetObstruction() bool {
	a := read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
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

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}
