# This is a multi-stage Dockerfile. The first part executes a build in a Golang
# container, and the second retrieves the binary from the build container and
# inserts it into a "scratch" image.

# Part 1: Compile the binary in a containerized Golang environment
#
FROM golang:1.13 as test

WORKDIR /go/bin/

COPY . /go/src/github.com/clockworksoul/smudge

RUN go test -v github.com/clockworksoul/smudge


# Part 2: Compile the binary in a containerized Golang environment
#
FROM golang:1.13 as build

WORKDIR /go/bin/

COPY . /go/src/github.com/clockworksoul/smudge

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o smudge github.com/clockworksoul/smudge/smudge


# Part 3: Build the Smudge image proper
#
FROM scratch as image

COPY --from=build /go/bin/smudge .

EXPOSE 9999

CMD ["/smudge"]
