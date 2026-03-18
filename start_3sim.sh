#!/bin/bash

# Start simulators in new terminal windows
gnome-terminal --title="Sim 1" -- bash -c "simelevatorserver --port 15657; exec bash"
gnome-terminal --title="Sim 2" -- bash -c "simelevatorserver --port 15656; exec bash"
gnome-terminal --title="Sim 3" -- bash -c "simelevatorserver --port 15655; exec bash"

# timeout /t 2 becomes sleep 2 in Linux
sleep 2

# Build the Go program (removed the .exe extension)
go build -o heis main.go

# Start the elevators in new terminal windows
gnome-terminal --title="Heis 1" -- bash -c "./heis -port 15657; exec bash"
gnome-terminal --title="Heis 2" -- bash -c "./heis -port 15656; exec bash"
gnome-terminal --title="Heis 3" -- bash -c "./heis -port 15655; exec bash"
