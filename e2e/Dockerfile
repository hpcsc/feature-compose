FROM golang:1.27.1 AS builder
# go.mod can ask for a newer Go patch release than the image has
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
# the test of a new repo runs task test:unit in it, which runs gotestsum, and the test of a CLI repo makes a
# GoReleaser snapshot of it, at the version of goreleaser in the mise.toml of this repo
RUN CGO_ENABLED=0 go install github.com/go-task/task/v3/cmd/task@v3.53.1 \
    && CGO_ENABLED=0 go install gotest.tools/gotestsum@v1.13.0 \
    && CGO_ENABLED=0 go install github.com/goreleaser/goreleaser/v2@v2.18.2
COPY . .
ARG BUILD_TAG=v0.1.0
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/hpcsc/feature-compose/internal/version.releaseTag=${BUILD_TAG}" \
    -o /feature-compose ./cmd/feature-compose

FROM node:24-trixie-slim
# python3, make and g++ build node-pty when no prebuilt node-pty fits the platform. g++ also brings the
# C compiler that the -race flag of a new repo's task test:unit needs. GoReleaser stops when no git is on
# the PATH, also outside a git repository.
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 make g++ git \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /go/bin/task /go/bin/gotestsum /go/bin/goreleaser /usr/local/bin/
ENV PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=auto
WORKDIR /e2e
COPY e2e/package.json e2e/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install --no-audit --no-fund; fi
COPY e2e ./
COPY catalog /catalog
COPY --from=builder /feature-compose /usr/local/bin/feature-compose
ARG BUILD_TAG=v0.1.0
ENV EXECUTABLE=/usr/local/bin/feature-compose BUILD_TAG=${BUILD_TAG} CATALOG_DIR=/catalog
CMD ["npx", "vitest", "run"]
