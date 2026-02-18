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
var topFloor int = _numFloors - 1
var _mtx sync.Mutex
var _conn net.Conn

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

type Elevator struct {
	OrderList    [4][numButtons]OrderStatus
	CabBackupMap map[string][4]OrderStatus
	Floor        int
	Retning      MotorDirection
	PrevRetning  MotorDirection
	DoorOpen     bool
	AliveNodes   map[string]bool
	ID           string
	MsgCount     int
}

type ElevatorStatus struct { //det som sendes, health checks
	SenderID     string
	CurrentFloor int
	Direction    int
	OrderList    [4][3]OrderStatus
	CabBackupMap map[string][4]OrderStatus
	MsgID        int //For å holde styr på rekkefølge, forkaste gamle meldinger
	DoorOpen     bool
}

type OrderStatus int

const (
	Order_Inactive = 0 // bruker int, kan eventuelt bruke veldig forskjellieg verdier for å gjøre robust mot "cosmic ray bitflip"
	Order_Pending  = 1 // UDP har vist checksum så mulig irellevant, kanskje bruke 0 til unknown siden det er default value for int?
	Order_Active   = 2
)

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

func (e *Elevator) UpdateElevatorOrder(btn ButtonEvent) {
	e.OrderList[btn.Floor][btn.Button] = Order_Pending
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
		for j := 0; j < numButtons; j++ {
			if e.OrderList[i][j] == Order_Active {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) HasOrderBelow() bool {
	for i := e.Floor - 1; i >= 0; i-- {
		for j := 0; j < numButtons; j++ {
			if e.OrderList[i][j] == Order_Active {
				return true
			}
		}

	}
	return false
}

func (e *Elevator) FloorOrder() bool {
	for i := 0; i < numButtons; i++ {
		if e.OrderList[e.Floor][i] == Order_Active {
			return true
		}
	}
	return false
}

func (e *Elevator) ActiveOrders() bool { //needed for PollFloorSensor
	for i := 0; i < _numFloors; i++ {
		for j := 0; j < numButtons; j++ {
			if e.OrderList[i][j] == Order_Active {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) ClearOrderFloor() { // mulig ikke lur måte å gjøre det på
	for i := 0; i < numButtons; i++ {
		if e.OrderList[e.Floor][i] == Order_Active {
			e.OrderList[e.Floor][i] = Order_Inactive
			e.SetButtonLamp(ButtonType(i), e.Floor, false)
		}

	}
}

func (e *Elevator) ClearOrderHallBtn() { // mulig ikke lur måte å gjøre det på
	for i := 0; i < numButtons-1; i++ {
		if e.OrderList[e.Floor][i] == Order_Active {
			e.OrderList[e.Floor][i] = Order_Inactive
			e.SetButtonLamp(ButtonType(i), e.Floor, false)
		}

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

func (e *Elevator) CabInit(ID string) {
	for GetFloor() != 0 {
		e.SetMotorDirection(MD_Down)
		time.Sleep(_pollRate)
	}
	e.SetMotorDirection(0)
	e.Floor = 0
	e.PrevRetning = 0
	e.Retning = 0
	e.SetDoorOpenLamp(false)
	e.AliveNodes = make(map[string]bool)
	e.CabBackupMap = make(map[string][4]OrderStatus)
	e.ID = ID
	e.MsgCount = 0
}

func (e *Elevator) DoorTimer(SendDone chan<- bool) {
	time.Sleep(3 * time.Second)
	SendDone <- true
}

func (e *Elevator) StoppFloor() {
	e.SetMotorDirection(0)
	e.DoorOpen = true
	e.SetDoorOpenLamp(true)
	e.ClearOrderFloor()
}

func (e *Elevator) ExecuteOrder() { // må kanskje forkaste hele denne til fordel for en GoToFloor funksjon, siden cost funksjon assigner ordre til heis
	switch {
	case e.FloorOrder():
		switch {
		case e.OrderList[e.Floor][2] == Order_Active: // knapp cab
			e.StoppFloor()
		case (e.Retning == 1) && (e.OrderList[e.Floor][0] == Order_Active): //på tur oppover og knapp hall opp
			e.StoppFloor()
		case e.Retning == -1 && (e.OrderList[e.Floor][1] == Order_Active): // tur nedover knapp hall ned
			e.StoppFloor()
		case e.Retning == 0 && ((e.OrderList[e.Floor][1] == Order_Active) || (e.OrderList[e.Floor][0] == Order_Active)): // står i ro, hall up/down åpen dør
			e.StoppFloor()
		case (e.Retning == -1) && (e.OrderList[e.Floor][0] == Order_Active) && (!e.HasOrderBelow()):
			e.StoppFloor()
		case (e.Retning == 1) && (e.OrderList[e.Floor][1] == Order_Active) && (!e.HasOrderAbove()):
			e.StoppFloor()

		default:
			e.SetMotorDirection(0) // mulig redundant
		}

	case e.HasOrderAbove():
		switch {
		case e.Retning == 1: // Har odre over er på tur opp -> fortsett opp
			e.SetMotorDirection(1)

		case e.Retning == -1: // HAr odre oveer er på tur ned -> fortsett ned
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 1) && e.Floor != topFloor: // Har ordre over, stoppet i etasje, var på tur opp og er ikke i toppetasjen -> kjør opp
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.PrevRetning == -1) && (!e.HasOrderBelow()):
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.PrevRetning == -1) && e.Floor != 0: // Har ordre over, var på tur ned stopped i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning != -1) && e.Floor != topFloor: // Har ordre over, stoppet i etasje, var ikke på tur og er ikke i toppetasjen -> kjør opp
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.Floor == 0):
			e.SetMotorDirection(1)

		default:
			e.SetMotorDirection(0)
		}

	case e.HasOrderBelow():
		switch {
		case e.Retning == 1: // Har odre under er på tur opp -> fortsett opp
			e.SetMotorDirection(1)

		case e.Retning == -1: // HAr odre oveer er på tur ned -> fortsett ned
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 1) && (!e.HasOrderAbove()): // står i ro, var på tur opp, har ikke odre over, men har under -> kjør ned
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.PrevRetning == 1) && e.Floor != topFloor: // Har ordre under, stoppet i etasje, var på tur opp og er ikke i toppetasjen -> kjør opp
			e.SetMotorDirection(1)

		case (e.Retning == 0) && (e.PrevRetning != 1) && e.Floor != 0: // Har ordre under, var ikke på tur opp, stoppet i en etasje, ikke bunn etasje-> kjør nedover
			e.SetMotorDirection(-1)

		case (e.Retning == 0) && (e.Floor == topFloor): // mulig redundant
			e.SetMotorDirection(-1)

		default:
			e.SetMotorDirection(0)
		}

	default:
		e.SetMotorDirection(0)
	}

}

func (e *Elevator) SteinSaksPapir(Node ElevatorStatus) { //Utfører steinsakspapir algebra
	for i := 0; i < _numFloors; i++ {
		for j := 0; j < 2; j++ {
			switch {
			case (e.OrderList[i][j] == Order_Inactive) && (Node.OrderList[i][j] == Order_Pending): // var inaktiv, får pending fra annen node = pending
				e.OrderList[i][j] = Order_Pending
			case (e.OrderList[i][j] == Order_Pending) && ((Node.OrderList[i][j] == Order_Pending) || (Node.OrderList[i][j] == Order_Active)): // Ordre er pending, får enten pending eller aktiv fra annen node -> aktiv
				e.OrderList[i][j] = Order_Active
				e.SetButtonLamp(ButtonType(j), i, true) // noe av det dummeste jeg har sett, caste i som er en int til buttontype som er en int
			case (e.OrderList[i][j] == Order_Active) && (Node.OrderList[i][j] == Order_Inactive): // Ordre er aktiv, blir utført annen node->satt inaktiv der = inaktiv her
				e.OrderList[i][j] = Order_Inactive
				e.SetButtonLamp(ButtonType(j), i, false)
			default: // legge til noe her? Usikker hva default case burde være
				continue
			}
		}
	}
	//skru på lamper cab orders, aktiver de basert på å sjekke map fra andre elev og egen orderlist
	CabBackup := Node.CabBackupMap[e.ID]
	for k := 0; k < _numFloors; k++ {
		switch {
		case (e.OrderList[k][2] == Order_Pending) && CabBackup[k] == Order_Active:
			e.OrderList[k][2] = Order_Active
			e.SetButtonLamp(ButtonType(2), k, true)

		default:
			continue
		}
	}
}

func (e *Elevator) CabBackupFunc(Node ElevatorStatus) {
	CabBackup := e.CabBackupMap[Node.SenderID]

	for k := 0; k < _numFloors; k++ {
		incomingCabstate := Node.OrderList[k][2]
		currentBackupState := CabBackup[k]
		switch {
		case (currentBackupState == Order_Inactive) && (incomingCabstate == Order_Pending):
			CabBackup[k] = Order_Pending

		case (currentBackupState == Order_Pending) && (incomingCabstate == Order_Pending || incomingCabstate == Order_Active):
			CabBackup[k] = Order_Active

		case (currentBackupState == Order_Active) && (incomingCabstate == Order_Inactive):
			CabBackup[k] = Order_Inactive

		default:
			continue
		}

	}
	e.CabBackupMap[Node.SenderID] = CabBackup
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
