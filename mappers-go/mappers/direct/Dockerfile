
FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY ./bin/direct kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge

ENTRYPOINT ["/kubeedge/direct", "--v", "5"]
