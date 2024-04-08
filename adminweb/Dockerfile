# Start from a Go-base image
FROM golang:latest as builder

# Work folder in container
WORKDIR /go/src/app

# Copy to containern
COPY . .

# Build app
#mount=type=cache,target=/root/.cache/go-build --
#--mount=type=cache,target=/root/.cache/go-build     
RUN GOOS=linux GOARCH=amd64 go build -v -o bin/vc_web1 -ldflags "-w -s --extldflags '-static'" ./cmd/main.go

# Install Node.js and npm
RUN apt-get update && apt-get install -y nodejs npm

# Run npm install for uuid-random (used in javascript to create uuid to tag.id)
RUN npm install uuid-random

# OS
FROM debian:bookworm-slim

WORKDIR /

# Copy the compiled app and HMTL files from previous build step
COPY --from=builder /go/src/app/bin/vc_web1 /vc_web1
COPY --from=builder /go/src/app/assets /assets/

# Expose port 8080
EXPOSE 8080

# Run webservern
CMD ["./vc_web1"]

# build image
# docker build --tag docker.sunet.se/dc4eu/web1:latest .

# run
# docker run -d -p 8090:8080 vc-web1:latest // :8090 temporärt för att inte krocka med vc containers så kan behöva ta hänsyn till det vid testkörning)

#Browse
#http://localhost:8090 //:8090 temporärt