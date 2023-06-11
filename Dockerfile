FROM golang:1.20 as build

WORKDIR /workdir
COPY go.* /workdir/
RUN go mod download
COPY . /workdir/

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix cgo -o tobab .


FROM gcr.io/distroless/static-debian11
COPY --from=build /workdir/tobab /tobab
CMD ["/tobab"]
