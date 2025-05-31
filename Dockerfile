FROM alpine:3.22
COPY ./bin/k8nskel /k8nskel
ENTRYPOINT /k8nskel
