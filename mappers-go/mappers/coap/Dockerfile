
FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY ./bin/coap kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge

ENTRYPOINT ["/kubeedge/coap", "--v", "5"]
