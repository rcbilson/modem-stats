FROM golang:1.23 AS build-monitor

WORKDIR /src
COPY go.mod go.sum .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \ 
    go mod download && go mod verify
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \ 
    go build

FROM telegraf:1.31 AS builder

ADD ./docker/entrypoint-msh.sh /entrypoint-msh.sh
RUN chmod +x /entrypoint-msh.sh

RUN mkdir -p /etc/telegraf.d/ /etc/template/
ADD ./docker/telegraf.conf /etc/template/

COPY --from=build-monitor /src/modem-stats /modem-stats

# We build from scratch as to remove all the volume and exposures from the
# source Telegraf Docker image. Since we don't have any state or listeners,
# this is "okay" to do.
FROM scratch

COPY --from=builder / /

ENTRYPOINT /entrypoint-msh.sh
