# DP-230-CRAWLER

Service for getting endpoints for provided site by extracting and visiting links from page bodies. 
Links are received from Kafka topic named ```API-Service-Message```.

Topic name and other parameters can be set as environment variables. Defaults are
```
CRAWLER_DEFAULT_TIMEOUT=60
CRAWLER_NUM_OF_THREADS=50
CRAWLER_MAX_DEPTH=3
KAFKA_URL=kafka:9092
KAFKA_TOPIC_API=API-Service-Message
```
Service consumes messages with value payload in JSON, formatted as follows:
```
{ 
    "id":"main-task-1", 
    "url":"http://httpstat.us/",  
    "forwardTo":[ "SQLI-check", "LFI-check" ] 
}
```
where ```id``` is main task ID, ```url``` is URL to work with, ```forvardTo``` - test-service topics to send results to.
Topic names for consuming test services are:
```
SQLI-check
BA-check
XSS-check
LFI-check
```
Service can be pulled from DockerHub as ```docker pull dmytrothr/parabellum.crawler:latest```

To produce payload for this service you can run ```kafka-console-producer.sh --topic API-Service-Message --broker-list localhost:9092``` directly in your kafka container (the Kafka-container itself for this service could be run from [local-compose.yml](https://github.com/ITA-Dnipro/Dp-230-Crawler/blob/main/local-compose.yml) by this command ```docker-compose -f local-compose.yml up -d```
Hints about how to produce/consume Kafka topics for yourself could be found here https://kafka.apache.org/quickstart
