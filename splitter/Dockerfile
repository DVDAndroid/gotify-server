FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags=-s -o /splitter .

FROM scratch
COPY --from=build /splitter /splitter
ENTRYPOINT ["/splitter"]
