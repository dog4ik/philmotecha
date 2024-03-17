FROM golang:1.22

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN mkdir /usr/local/bin/server
RUN go build -v -o /usr/local/bin/server ./...

ENV port=6969
ENV DATABASE_URL="postgresql://postgres:123@db:5432/philmotecha"
ENV ACCESS_TOKEN=279eba547ba8a8ce185562d2a933987bec16f3c50bc97e49835651faef57959fdd7a576abb4d8705c3459a4285538df42bbd4dc41be00a245e2162e100e2ba73
ENV REFRESH_TOKEN=de6f8fe0105b40a6a13f350f1750143ed10d1f2ae608260fb733da866e30d0cdba94f5b9a61af126fe9812b7e2ce4debb68b64f847fc7e362de3773aee297a1c

CMD ["/usr/local/bin/server/philmotecha"]
