FROM golang:trixie AS builder

WORKDIR /build

COPY . .

RUN go build . 

FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /bin 

COPY --from=builder /build/lmproxy .

ENTRYPOINT [ "lmproxy" ]
