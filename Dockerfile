FROM golang:1.6.0

# project name
ENV PROJECT_PATH=/go/src/github.com/brocaar/lora-semtech-bridge

# set PATH
ENV PATH=$PATH:$PROJECT_PATH/bin

# install tools
RUN go get github.com/golang/lint/golint
RUN go get github.com/kisielk/errcheck

# setup work directory
RUN mkdir -p $PROJECT_PATH
WORKDIR $PROJECT_PATH

# copy source code
COPY . $PROJECT_PATH

# install requirements
RUN go get -t ./...
