
package main

import (
	"strconv"
	"time"
	"fmt"
	
	"github.com/influxdata/influxdb1-client/v2"
	"github.com/kvantetore/go.hue"
)

type InfluxSettings struct {
	serverURL string
	username string
	password string
	dbName string
	measurementName string
}

func findRoom(rooms []*hue.Group, light *hue.Light) (*hue.Group, error) {
	for _, room := range rooms {
		for _, lightIndex := range room.Lights {
			if light.Id == lightIndex {
				return room, nil
			}
		}
	}

	return nil, fmt.Errorf("No room found for light %v (%v)", light.Id, light.Name)
}

 //StoreSensorData saves the current state of the sensors to an influxdb measurement
func StoreSensorData(settings InfluxSettings, lights []*hue.Light, rooms []*hue.Group) error {
	//create influx client
	cli, err := client.NewHTTPClient(client.HTTPConfig {
		Addr: settings.serverURL,
		Username: settings.username,
		Password: settings.password,
	})
	if err != nil {
		return fmt.Errorf("Failed to create HTTP Client, %v", err)
	}
	defer cli.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  settings.dbName,
		Precision: "s",
	})
	if err != nil {
		return fmt.Errorf("Error creating batch points %v", err)
	}

	//create points
	currentTime := time.Now();
	for _, light := range lights {
		room, err := findRoom(rooms, light)
		if err != nil {
			return err
		}

		lightAttributes, err := light.GetLightAttributes()
		if err != nil {
			return fmt.Errorf("Error getting light attributes for light %v, %v", light.Name, err)
		}

		// Create a point and add to batch
		tags := map[string]string{
			"light_name": light.Name,
			"room_name": room.Name,
		}

		//tag by color name
		if (lightAttributes.State.ColorMode != "") {
			tags["color_mode"] = lightAttributes.State.ColorMode
		} else {
			tags["color_mode"] = "white"
		}
		
		//tag by state
		if !lightAttributes.State.Reachable {
			tags["state"] = "unreachable"
		} else if lightAttributes.State.On {
			tags["state"] = "on"
		} else {
			tags["state"] = "off"
		}

		//parse light and room ids as numbers. This allows us to use the ids as field
		//values, which in turn allows us to create queries like "select distinct(light_id) ..."
		lightIdInt, err := strconv.ParseInt(light.Id, 10, 8)
		if err != nil {
			return fmt.Errorf("Unable to parse light id '%v' as int, %v", light.Id, err)
		}

		roomIdInt, err := strconv.ParseInt(room.Id, 10, 8)
		if err != nil {
			return fmt.Errorf("Unable to parse room id '%v' as int, %v", room.Id, err)
		}

		fields := map[string]interface{}{
			"light_id": lightIdInt,
			"room_id": roomIdInt,
		}

		if lightAttributes.State.Reachable {
			//state and brightness
			if lightAttributes.State.On {
				fields["brightness"] = lightAttributes.State.Bri
			}

			//color temperature
			if lightAttributes.State.Ct > 0 {
				fields["color_temperature"] = int(1000000 / lightAttributes.State.Ct)
			}

			//color
			if lightAttributes.State.ColorMode == "xy" || lightAttributes.State.ColorMode == "hs" {
				fields["hue"] = lightAttributes.State.Hue / 255
				fields["saturation"] = lightAttributes.State.Sat

				if len(lightAttributes.State.Xy) == 2 {
					fields["color_x"] = lightAttributes.State.Xy[0]
					fields["color_y"] = lightAttributes.State.Xy[1]
				}
			}
		}

		pt, err := client.NewPoint(settings.measurementName, tags, fields, currentTime)
		if err != nil {
			return fmt.Errorf("Error creating new point, %v", err)
		}
		bp.AddPoint(pt)
	}

	//Write the batch
	if err := cli.Write(bp); err != nil {
		return fmt.Errorf("error writing points, %v", err)
	}

	return nil;
}
