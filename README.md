## App description

This app implements task for WATTx Software Engineer Challenge: Heating Control. To see the full task description please visit [this link](https://github.com/WATTx/code-challenges/blob/master/software-engineer-challenge-heating-control.md).

### Implementation overview:
There are three main components which implement heating control system: 
- controller
- sensors
- valve

Sensors periodically measure current temperature and send data to mqtt broker topic. The controller gets this data, calculates average temperature and defines valve level openness.
Controller sends valve openness value to specific mqtt topic, valve gets it. It is assumed that the temperature read by sensors will be changed according to the valve level.
Number of sensors and measure timeout is configurable.
At the start of the application, sensors initialize with random temperature in float range [0.1;100) degrees.
Then after processing data, the temperature on each sensor changes on some percent value depending on valve openness.
Current task solution based on a statement that valve level below 50 decreases temperature; the temperature increases at valve level above 50. For example, if current temperature is 30 degrees, and the valve openness set to 30, then on next iteration a sensor will get temperature at 20% below, 30-6=24. So other sensors do.
So asa the time passes, the average temperature will be aim to the temperature level set in the config file.
As a destabilizing factor, the temperature read by sensor may decrease on [0;1] degree. With that, average temperature won't be stable by time, and the valve level will always be changed, aiming to the required temperature level.
When temperature measured by a sensor runs below 0 degrees, and valve level is above 50, the temperature increases.   

In the log output you can see:
- current temperature read by the sensors ([sensor-X])
- controller: receiving data, define average temperature, set valve level ([ctrl])
- mqtt publishing data ([mqtt])
- valve receiving data ([valve])
- average temperature history, for example:

`heatingcontrol_1  | 2021/07/16 11:48:30 [ctrl] Average temperature history: [41.3 24.4 23.4 23.2 23 22.6 22 24 23.4 22.8 22.2 21.6 23.8 23.2 22.6]`

Required temperature level set to 22.5 degrees at start, and as you can see, average temperature runs from 41 degrees to 21.6, then above 23, the again aiming to 22.5 degree.

### Hints

Mosquitto broker might be use either from separate local docker container, or as a cloud service, for example `broker.hivemq.com`.

Mosquitto host and port set in config override by env variables `MQTT_BROKER`, `MQTT_PORT`.


### Docker compose
To build and run application in docker-compose:
`docker-compose up --build --force-recreate`

### Useful Docker commands
build app in docker:

`docker build -t heatingcontrol .`

run mosquitto:

`docker run -it -p 1883:1883 --name=mosquitto  toke/mosquitto`

run container:

`docker start mosquitto`

stop mosquitto:

`docker stop mosquitto`

view logs:
`docker logs -f mosquitto`

# TODO
- tests 
- motion sensor
- many rooms
