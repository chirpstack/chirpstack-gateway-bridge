FROM golang:1.10-alpine AS development

ENV PROJECT_PATH=/go/src/github.com/brocaar/lora-gateway-bridge
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=0
ENV GO_EXTRA_BUILD_ARGS="-a -installsuffix cgo"

RUN apk add --no-cache ca-certificates make git bash

RUN mkdir -p $PROJECT_PATH
COPY . $PROJECT_PATH
WORKDIR $PROJECT_PATH

RUN make requirements
RUN make

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=development /go/src/github.com/brocaar/lora-gateway-bridge/build/lora-gateway-bridge .
ENTRYPOINT ["./lora-gateway-bridge"]
