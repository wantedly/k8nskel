FROM alpine:3.17
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
