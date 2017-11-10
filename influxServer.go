
package main

import (
	"time"
	"fmt"
	
	"github.com/influxdata/influxdb/client/v2"
	"github.com/stefanwichmann/go.hue"
)

type InfluxSettings struct {
	serverURL string
	dbName string
	measurementName string
}

 //StoreSensorData saves the current state of the sensors to an influxdb measurement
func StoreSensorData(settings InfluxSettings, lights []*hue.Light) error {
	//create influx client
	cli, err := client.NewHTTPClient(client.HTTPConfig {
		Addr: settings.serverURL,
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
		lightAttributes, err := light.GetLightAttributes()
		if err != nil {
			return fmt.Errorf("Error getting light attributes for light %v, %v", light.Name, err)
		}


		// Create a point and add to batch
		tags := map[string]string{
			"light_name": light.Name,
			"light_id": light.Id,
		}
		if (lightAttributes.State.ColorMode != "") {
			tags["color_mode"] = lightAttributes.State.ColorMode
		} else {
			tags["color_mode"] = "white"
		}

		fields := map[string]interface{}{
		}

		if lightAttributes.State.Reachable {
			//state and brightness
			if lightAttributes.State.On {
				fields["state"] = 1
				fields["brightness"] = lightAttributes.State.Bri
			} else {
				fields["state"] = 0
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
		} else {
			fields["state"] = 0
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
