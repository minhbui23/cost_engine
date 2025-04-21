# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Install build dependencies, including kernel headers
RUN apk update && \
    apk add --no-cache make gcc g++ leveldb-dev linux-headers

# Set up environment variables
ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Install dependencies and build the application
RUN make deps && \
    make build && \
    make install

# Stage 2: Create the lightweight runtime image
FROM alpine:latest

# Install runtime dependencies for LevelDB
RUN apk add --no-cache leveldb

# ARGs for metadata
ARG GIT_URL=unspecified
ARG GIT_COMMIT=unspecified
ARG GIT_BRANCH=unspecified
ARG DATE=unspecified
ARG TAG=latest

# Add metadata labels
LABEL org.opencontainers.image.vendor="SoC.one" \
      org.opencontainers.image.licenses="Proprietary" \
      org.opencontainers.image.description="SoC.one Payment Network" \
      org.opencontainers.image.documentation="https://soc.one" \
      org.opencontainers.image.source=$GIT_URL \
      org.opencontainers.image.revision=$GIT_COMMIT \
      org.opencontainers.image.branch=$GIT_BRANCH \
      org.opencontainers.image.date=$DATE \
      org.opencontainers.image.version=$TAG

# Copy the compiled binary from the builder stage
COPY --from=builder /app/build/streampayd /usr/bin/streampayd

# Set the entrypoint command to run the application
ENTRYPOINT ["/usr/bin/streampayd"]
