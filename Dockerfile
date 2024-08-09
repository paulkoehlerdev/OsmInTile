FROM golang:1.22.6-bullseye AS build

WORKDIR /app

COPY . .

RUN go get ./...

RUN go build -o /build/osmintile ./cmd/osmintile

FROM scratch

COPY --from=build /app/build/osmintile /bin/osmintile

CMD ["/bin/osmintile"]