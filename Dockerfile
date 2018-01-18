# This is a multi-stage Dockerfile. The first part executes a build in a Golang
# container, and the second retrieves the binary from the build container and
# inserts it into a "scratch" image.

# Part 1: Compile the binary in a containerized Golang environment
#
FROM golang:1.7 as build

MAINTAINER Matt Titmus <matthew.titmus@gmail.com>

WORKDIR /go/bin/

COPY . /go/src/github.com/clockworksoul/smudge

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o smudge github.com/clockworksoul/smudge/smudge

# Part 2: Build the Smudge image proper
#
FROM scratch

MAINTAINER Matt Titmus <matthew.titmus@gmail.com>

COPY --from=build /go/bin/smudge .

EXPOSE 9999

CMD ["/smudge"]
