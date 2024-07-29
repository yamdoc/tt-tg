FROM golang:alpine3.20 AS build

WORKDIR /build

COPY go.mod go.sum /build/
RUN go mod download

COPY cmd/tt-tg /build/cmd/tt-tg
COPY internal/tt-tg /build/internal/tt-tg
RUN go build -ldflags '-extldflags "-static"' -tags netgo,osusergo -o tt_tg ./cmd/tt-tg


FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ffmpeg
COPY --from=build /build/tt_tg /app/

ENV TT_TG_CONFIG=config/config.yaml
ENV TT_TG_POLL_RATE=480
ENV TT_TG_API_TG=
ENV TT_TG_API_TT=

ENTRYPOINT "./tt_tg" "-api-tg" ${TT_TG_API_TG} "-api-tt" ${TT_TG_API_TT} "-cfg" ${TT_TG_CONFIG} "-poll-m" ${TT_TG_POLL_RATE}