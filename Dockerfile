FROM alpine:3.19
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
