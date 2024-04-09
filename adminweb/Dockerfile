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

## Install Node.js and npm
#RUN apt-get update && apt-get install -y nodejs npm
#
## Run npm install for uuid-random
#RUN npm install uuid-random


# OS
FROM debian:bookworm-slim
#FROM ubuntu:latest

WORKDIR /

RUN apt-get update && apt-get install -y curl procps iputils-ping less
RUN rm -rf /var/lib/apt/lists/*

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
# docker run -d -p 8080:8080 vc-web1:latest

#Browse
# http://http://172.16.50.12:8080/ (if docker-compose/make) or  http://localhost:8080

# Inne i container
#docker exec -ti vc_web1 bash