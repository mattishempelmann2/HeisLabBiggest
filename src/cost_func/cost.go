package cost

import (
	"encoding/json"
	"fmt"
	"heis/src/elevio"
	"os/exec"
	"runtime"
)

var dirMap = map[int]string{
	1:  "up",
	-1: "down",
	0:  "stop",
}

var OrderBoolMap = map[elevio.OrderStatus]bool{
	elevio.Order_Inactive: false,
	elevio.Order_Pending:  false,
	elevio.Order_Active:   true,
}

type HRAElevState struct {
	Behavior    string  `json:"behaviour"`
	Floor       int     `json:"floor"`
	Direction   string  `json:"direction"`
	CabRequests [4]bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [4][2]bool              `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func makeHRAElevState(Node any) HRAElevState {
	elevState := &HRAElevState{}
	switch Nodetype := Node.(type) {
	case elevio.Elevator:
		elevState.Behavior = Nodetype.Behaviour
		elevState.Floor = Nodetype.Floor
		elevState.Direction = dirMap[int(Nodetype.Direction)]
		for floor := 0; floor < 4; floor++ {
			elevState.CabRequests[floor] = OrderBoolMap[Nodetype.OrderListCab[floor]]
		}

	case elevio.ElevatorStatus:
		elevState.Behavior = Nodetype.Behaviour
		elevState.Floor = Nodetype.CurrentFloor
		elevState.Direction = dirMap[int(Nodetype.Direction)]
		for floor := 0; floor < 4; floor++ {
			elevState.CabRequests[floor] = OrderBoolMap[Nodetype.OrderListCab[floor]]
		}
	default:
		fmt.Printf("Unsupported Node type")
	}
	return *elevState
}

func MakeHRAInput(localNode elevio.Elevator, otherNodes map[string]elevio.ElevatorStatus) HRAInput {
	HRAInput := &HRAInput{}
	HRAInput.States = make(map[string]HRAElevState)

	for floor := 0; floor < 4; floor++ {
		for button := 0; button < 2; button++ {
			HRAInput.HallRequests[floor][button] = OrderBoolMap[localNode.OrderListHall[floor][button]]
		}
	}

	HRAInput.States[localNode.ID] = makeHRAElevState(localNode)
	for id, status := range otherNodes {
		HRAInput.States[id] = makeHRAElevState(status)
	}

	return *HRAInput
}

func CostFunc(input HRAInput) map[string][4][2]bool {

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	case "darwin":
		hraExecutable = "hall_request_assigner_mac"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return nil
	}

	ret, err := exec.Command("./cost_fns/hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][4][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return nil
	}
	return *output

	/*fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}*/
}
