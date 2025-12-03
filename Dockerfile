FROM alpine:3.23
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
