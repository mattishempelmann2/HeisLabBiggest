@echo off
start "Sim 1" cmd /k ".\simelevatorserver.exe --port 15657"
start "Sim 2" cmd /k ".\simelevatorserver.exe --port 15656"
start "Sim 3" cmd /k ".\simelevatorserver.exe --port 15655"
timeout /t 2
go build -o heis.exe main.go



start "Heis 1" cmd /k ".\heis.exe -port 15657"
start "Heis 2" cmd /k ".\heis.exe -port 15656"
start "Heis 3" cmd /k ".\heis.exe -port 15655"