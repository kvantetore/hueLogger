package main

import (
	"os"
	"fmt"
	"bufio"
	"time"
	"github.com/kvantetore/go.hue"
)

const (
	influxServer = "http://pi:8086"
	influxDb = "home"
	lightMeasurement = "lights"
)

func connectToBridge() (bridge *hue.Bridge, err error) {
	bridges, err := hue.DiscoverBridges(false)
	if err != nil {
		fmt.Printf("Unable to find Hue Portal %v", err)
		os.Exit(1)
	}
	bridge = &bridges[0]
	fmt.Printf("Found bridge %+v\n", bridge)

	hueUsername := os.Getenv("HUE_USERNAME")
	if hueUsername != "" {
		bridge.Username = hueUsername
		return bridge, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Environment variable HUE_USERNAME not set, creating new username...")
	fmt.Println("Please press the link button on your hub, then press [enter] to continue.")
	reader.ReadLine()

	err = bridge.CreateUser("Apokalypse Hue Logger")
	if err != nil {
		return nil, fmt.Errorf("Error creating user: %v", err)
	}

	fmt.Println("Connected to bridge, please set environment variable HUE_USERNAME to ", bridge.Username)
	return bridge, nil
}
 
func main() {
	bridge, err := connectToBridge()
	if err != nil {
		fmt.Println("Error connecting to bridge: ", err)
		os.Exit(1)
	}

	influxSettings := InfluxSettings {
		serverURL: influxServer,
		dbName: influxDb,
		measurementName: lightMeasurement,
	}	
	
	performMeasurement := func() {
		rooms, err := bridge.GetAllRooms()
		if err != nil {
			fmt.Printf("Error fetching rooms: %v\n", err)
			return
		}

		lights, err := bridge.GetAllLights()
		if err != nil {
			fmt.Printf("Error fetching lights, %v\n", err)
			return
		}

		err = StoreSensorData(influxSettings, lights, rooms)	
		if err != nil {
			fmt.Printf("Error light data %v\n", err)
		}

		fmt.Printf("%v lights stored\n", len(lights))
	}

	//create timer that runs measurement 
	interval := time.Minute * 1
	fmt.Printf("Running logger every %v...\n", interval)
	ticker := time.NewTicker(interval)
	for {
		performMeasurement()
		<- ticker.C
	}
	
}