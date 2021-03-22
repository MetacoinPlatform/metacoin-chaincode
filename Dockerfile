# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

ARG GO_VER=1.15.5
ARG ALPINE_VER=3.12

FROM golang:${GO_VER}-alpine${ALPINE_VER}

ENV CHAINCODE_ID metacoin_2.0.2:20686d14decf0f62800379503a3103f6119b61cbdbb07027b599c212776f7e36
ENV CHAINCODE_SERVER_ADDRESS 0.0.0.0:7049

WORKDIR /go/src/inblock.co/metacoin
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 7049
CMD ["metacoin"]
