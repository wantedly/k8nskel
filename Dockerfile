FROM alpine:3.21
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
