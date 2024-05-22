FROM alpine:3.20
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
