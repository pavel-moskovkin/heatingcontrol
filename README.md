run mosquitto:
`docker run -it -p 1883:1883 --name=mosquitto  toke/mosquitto`
run container:
`docker start mosquitto`
stop mosquitto:
`docker stop mosquitto`
logs:
`docker logs -f mosquitto`

---

docker build:
`docker build -t heatingcontrol .`

To run an application in docker-compose:
`docker compose up`

