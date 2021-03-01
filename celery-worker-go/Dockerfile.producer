FROM golang as build-stage
COPY go.mod /
COPY go.sum /
RUN go mod download
COPY producer.go /
RUN cd / && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o producer

FROM alpine
COPY --from=build-stage /producer /
CMD ["/producer"]