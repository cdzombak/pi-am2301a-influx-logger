# `pi-am2301a-influx-logger`

Logs temperature and humidity readings, on a regular basis, to InfluxDB from an AM2301/DHT21 sensor attached to a Raspberry Pi.

## Compiling

The DHT library uses C under the hood, so cross-compiling as with normal Go code doesn't work, and I haven't bothered to figure out how to fix it.

So: clone the repository to the Pi Zero W, and run `go build .` from the root of the repo. Building takes a while.

After compiling, I move the binary to `/usr/local/bin` and change its owner to `root`:
```
sudo chown root:root ./pi-am2301a-influx-logger
sudo mv ./pi-am2301a-influx-logger /usr/local/bin
```

## Connecting the Sensor

The AM2301a/DHT21 use a custom "one wire" interface. Connect the signal wire to a GPIO pin of the Pi. (I used GPIO 4, which is pin 16 on the Pi's GPIO header.)

Aside from that, connect the sensor's power and ground wires to the 5V supply (pin 2 or 4 of the GPIO header) and ground (pin 6 of the GPIO header).

## Usage

- `-gpio-pin int`: GPIO pin the sensor is connected to. (default `4`)
- `-humid-max int`: Maximum relative humidity reading. Readings outside this range are discarded. (default `100`)
- `-humid-min int`: Minimum relative humidity reading. Readings outside this range are discarded. (default `0`)
- `-influx-attempts uint`: Number of attempts to make to send a reading to InfluxDB. (default `3`)
- `-influx-bucket string`: InfluxDB bucket. Supply a string in the form `database/retention-policy`. For the default retention policy, pass just a database name (without the slash character). Required.
- `-influx-password string`: InfluxDB password.
- `-influx-server string`: InfluxDB server, including protocol and port, eg. `http://192.168.1.1:8086`. Required.
- `-influx-timeout int`: InfluxDB request timeout, in seconds. (default `3`)
- `-influx-username string`: InfluxDB username.
- `-log-interval int`: Logging interval, in seconds (log every X seconds). (default `60`)
- `-log-readings`: Log temperature/humidity readings to standard error.
- `-max-retries int`: Maximum number of attempts to read the DHT21/AM2301 sensor. (default `12`)
- `-measurement-name string`: InfluxDB measurement name. (default `temperature_humidity`)
- `-sensor-name string`: Value for the sensor_name tag in InfluxDB. Required.
- `-temp-max int`: Maximum temperature reading, in degrees C. Readings outside this range are discarded. (default `80`)
- `-temp-min int`: Minimum temperature reading, in degrees C. Readings outside this range are discarded. (default `-40`)

## Persistent Installation with systemd

Copy [`logger.service`](logger.service) to the `/etc/systemd/system` directory, and customize it as desired (adjusting the path to the logger binary, options passed to the program, and service name/description). Rename it to something else if you'd like.

Then:
```
sudo systemctl daemon-reload
sudo systemctl enable logger.service
sudo systemctl start logger.service
```

Verify the logger is running properly via `sudo journalctl -f -u logger.service`.

## License

MIT; see [`LICENSE`](LICENSE) in this repository.
