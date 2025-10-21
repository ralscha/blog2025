Code for blog post: https://blog.rasc.ch/2025/08/mqtt.html

docker run --name emqx-enterprise -p 1883:1883 -p 18083:18083 emqx/emqx-enterprise:5.10.0


docker build -t mqtt-protogen .
docker run -v "c:\w\ws\preblog\mqtt\schema:/app/schema" -v "c:\w\ws\preblog\mqtt\publisher:/app/publisher" mqtt-protogen