FROM golang:1.22.6-bullseye AS toolbox

WORKDIR /app

# install spatialite
RUN export DEBIAN_FRONTEND=noninteractive
RUN apt update
RUN apt install -y --no-install-recommends sqlite3 libsqlite3-mod-spatialite zlib1g-dev
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

CMD tail -f /dev/null

FROM golang:1.22.6-bullseye AS build

WORKDIR /app

COPY . .

RUN go get ./...

RUN go build -o /build/osmintile ./cmd/osmintile

FROM scratch

COPY --from=build /app/build/osmintile /bin/osmintile

CMD ["/bin/osmintile"]