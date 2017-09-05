FROM alpine:3.6
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
