FROM alpine:3.18
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
