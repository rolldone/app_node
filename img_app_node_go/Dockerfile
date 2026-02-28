# Multi-stage Dockerfile to build the Go binary and produce a small runtime image

FROM golang:1.25.3-alpine
ARG UID=1000
ARG GID=1000
RUN apk add --no-cache ca-certificates bash iputils busybox-extras nano wget
RUN echo 'export PATH=/usr/local/go/bin:$PATH' >> /etc/profile
RUN echo 'umask 002' >> /etc/profile

WORKDIR /app

# Create group and user with configurable UID/GID (defaults keep backward compatibility)
RUN addgroup -g ${GID} go \
 && adduser -D -u ${UID} -G go -s /bin/bash go \
 && addgroup go root

# Cache modules
# COPY go.mod go.sum ./
# RUN go mod download

# Install golang-migrate and swag (swag CLI & gin-swagger deps)
RUN go install -tags 'postgres mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
 && go install github.com/swaggo/swag/cmd/swag@latest

# Copy the rest of the repo and build
COPY . .

# Ensure repository files are owned by the `go` user (use build args UID/GID)
RUN chown -R ${UID}:${GID} /app || true

# Ensure gin-swagger and files packages are added to the module (after copying source)
RUN go get -u github.com/swaggo/gin-swagger github.com/swaggo/files || true

# Make migrate and swag globally available
RUN cp /go/bin/migrate /usr/local/bin/migrate || true \
 && cp /go/bin/swag /usr/local/bin/swag || true

# Generate swagger docs during build (produces /app/docs)
# If you prefer generating docs in CI instead, remove this line
RUN swag init -g cmd/main.go -o ./docs || true

# RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd

ENV PORT=8080
ENV PATH=/usr/local/go/bin:$PATH
EXPOSE 8080
CMD ["/out/app"]
