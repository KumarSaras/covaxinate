FROM golang:1.14.9-alpine AS builder
RUN mkdir /build
ADD go.mod go.sum /build/
ADD common /build/common
ADD slack /build/slack
WORKDIR /build
RUN go build ./common...
RUN cd slack && go build main.go 

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/slack /app/slack
WORKDIR /app/slack
CMD ["./main"]