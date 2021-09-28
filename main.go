package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/avast/retry-go"
	"github.com/d2r2/go-dht"
	"github.com/influxdata/influxdb-client-go/v2"
)

func main() {
	var logResults = flag.Bool("log-readings", false, "Log temperature/humidity readings to standard error.")
	var logEvery = flag.Int("log-interval", 60, "Logging interval, in seconds (log every X seconds).")
	// Influx settings:
	var influxServer = flag.String("influx-server", "", "InfluxDB server, including protocol and port, eg. 'http://192.168.1.1:8086'. Required.")
	var influxUser = flag.String("influx-username", "", "InfluxDB username.")
	var influxPass = flag.String("influx-password", "", "InfluxDB password.")
	var influxBucket = flag.String("influx-bucket", "", "InfluxDB bucket. Supply a string in the form 'database/retention-policy'. For the default retention policy, pass just a database name (without the slash character). Required.")
	var measurementName = flag.String("measurement-name", "temperature_humidity", "InfluxDB measurement name.")
	var sensorName = flag.String("sensor-name", "", "Value for the sensor_name tag in InfluxDB. Required.")
	var influxTimeoutSec = flag.Int("influx-timeout", 3, "InfluxDB request timeout, in seconds.")
	var influxAttempts = flag.Uint("influx-attempts", 3, "Number of attempts to make to send a reading to InfluxDB.")
	// Sensor settings:
	var maxRetries = flag.Int("max-retries", 12, "Maximum number of attempts to read the DHT21/AM2301 sensor.")
	var pin = flag.Int("gpio-pin", 4, "GPIO pin the sensor is connected to.")
	// Default min/max value are the min/max the sensor is specified to read:
	var tempMin = flag.Int("temp-min", -40, "Minimum temperature reading, in degrees C. Readings outside this range are discarded.")
	var tempMax = flag.Int("temp-max", 80, "Maximum temperature reading, in degrees C. Readings outside this range are discarded.")
	var humidMin = flag.Int("humid-min", 0, "Minimum relative humidity reading. Readings outside this range are discarded.")
	var humidMax = flag.Int("humid-max", 100, "Maximum relative humidity reading. Readings outside this range are discarded.")
	flag.Parse()
	if *influxServer == "" || *influxBucket == "" {
		fmt.Println("-influx-bucket and -influx-server must be supplied.")
		os.Exit(1)
	}
	if *sensorName == "" {
		fmt.Println("-sensor-name must be supplied.")
		os.Exit(1)
	}

	authString := ""
	if *influxUser != "" || *influxPass != "" {
		authString = fmt.Sprintf("%s:%s", *influxUser, *influxPass)
	}
	influxClient := influxdb2.NewClient(*influxServer, authString)
	influxTimeout := time.Duration(*influxTimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), influxTimeout)
	defer cancel()
	health, err := influxClient.Health(ctx)
	if err != nil {
		log.Fatalf("failed to check InfluxDB health: %v", err)
	}
	if health.Status != "pass" {
		log.Fatalf("InfluxDB did not pass health check: status %s; message '%s'", health.Status, *health.Message)
	}
	influxWriteApi := influxClient.WriteAPIBlocking("", *influxBucket)

	doUpdate := func() {
		atTime := time.Now()

		tempC, humidity, _, err := dht.ReadDHTxxWithRetry(dht.AM2302, *pin, false, *maxRetries)
		if err != nil {
			log.Printf("error: failed to read sensor for %s", atTime)
			return
		}
		if tempC < float32(*tempMin) || tempC > float32(*tempMax) {
			log.Printf("error: temperature reading %.1f degrees C is outside allowable range", tempC)
			return
		}
		if humidity < float32(*humidMin) || humidity > float32(*humidMax) {
			log.Printf("error: humidity reading %.1f%% is outside allowable range", humidity)
			return
		}

		tempF := tempC*1.8 + 32.0
		dewPoint := tempF - ((100.0 - humidity) * (9.0/25.0))

		if *logResults {
			log.Printf("temperature %.1f degF (%.1f degC); humidity %.1f%%; approx. dew point %.1f degF", tempF, tempC, humidity, dewPoint)
		}

		point := influxdb2.NewPoint(
			*measurementName,
			map[string]string{"sensor_name": *sensorName}, // tags
			map[string]interface{}{
				"temperature_f": tempF,
				"temperature_c": tempC,
				"humidity":      humidity,
				"dew_point_f":   dewPoint,
			}, // fields
			atTime,
		)
		if err := retry.Do(
			func() error {
				ctx, cancel := context.WithTimeout(context.Background(), influxTimeout)
				defer cancel()
				return influxWriteApi.WritePoint(ctx, point)
			},
			retry.Attempts(*influxAttempts),
		); err != nil {
			log.Printf("failed to write point to influx: %v", err)
		}
	}

	doUpdate()
	for {
		select {
		case <-time.Tick(time.Duration(*logEvery) * time.Second):
			doUpdate()
		}
	}
}
