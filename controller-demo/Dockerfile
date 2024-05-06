FROM golang:1.22 as builder

WORKDIR /app

COPY . .

ENV GO111MODULE=on GOPROXY=https://goproxy.cn,direct

RUN CGO_ENABLED=0 go build -o ingress-manager main.go

FROM alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/ingress-manager .

CMD [ "./ingress-manager" ]