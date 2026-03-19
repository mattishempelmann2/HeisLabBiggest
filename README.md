# HeisLabBiggest  

## Beskrivelse  Dette prosjektet implementerer et distribuert heissystem i Go, hvor flere heiser samarbeider over nettverk ved hjelp av UDP-kommunikasjon. Systemet håndterer bestillinger, kjører heisen og fordeler ordre mellom noder.  

## Forfattere  
* Mattis Jost Hempellman
* Magnus Fullu
* Lisa Radford
  
## Hvordan kjøre  
### 1. Start simulator  Åpne en terminal:  ``` cd "Sim apple silicon" ./SimElevatorServer --port 15657 ```  
### 2. Kjør programmet  Åpne en ny terminal:  ``` go run . -port=15657 ```  ⚠️ Porten må være lik begge steder.  
