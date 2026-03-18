#!/bin/bash

# Start simulators in new terminal windows
gnome-terminal --title="Server 2" -- bash -c "elevatorserver --port 15656; exec bash"

# timeout /t 2 becomes sleep 2 in Linux
sleep 2

# Build the Go program (removed the .exe extension)
go build -o heis main.go

# Start the elevators in new terminal windows
gnome-terminal --title="Heis 2" -- bash -c "./heis -port 15656; exec bash"

