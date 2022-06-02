FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -a -tags osusergo,netgo --ldflags '-extldflags "-f no-PIC -static"' .

FROM alpine

COPY --from=builder /app/mappend /usr/bin/

ENTRYPOINT ["mappend"]
