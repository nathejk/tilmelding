#
# DEV - BASE
FROM golang:1.25 AS api-dev

RUN mkdir -p /app/docker/bin
WORKDIR /app

# Persist the Go build cache in the api:/go volume so `go tool` doesn't rebuild
# the dev tools (staticcheck/gosec/govulncheck) from scratch on every start.
# The cache is toolchain-keyed, so a Go bump invalidates it automatically.
ENV GOCACHE=/go/.cache/go-build

ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; \
	apt -qq update && \
    apt install -qq -y --no-install-recommends \
        bash \
        inotify-tools && \
    apt -qq clean

COPY docker/init/api-dev /init
ENTRYPOINT ["/init", "api"]


#
# BASE
FROM api-dev AS base

# copy and download go mod dependencies
COPY go/go.mod go/go.sum ./
RUN go mod download

# copy src
COPY go/ .

#
# BUILD
FROM base AS build

# Disable Go workspaces for the production build: the build server has no
# sibling shared-go checkout, so dependencies must resolve from go.mod alone
# via the public module proxy. A stray go.work would otherwise break CI.
ENV GOWORK=off

# test and lint before building
RUN go test -timeout 60s ./... && \
    go tool staticcheck ./...

# build service
RUN GOARCH=amd64 CGO_ENABLED=1 GOOS=linux \
	go build -a -ldflags '-extldflags "-static"' -ldflags="-w -s" -o api nathejk.dk/cmd/api

## UI
FROM node:20.11.1-alpine3.19 AS ui-dev

RUN mkdir -p /app
WORKDIR /app
COPY vue /app

# Use the npm bundled with Node 20 (v10). The project has no `engines`
# constraint, and modern deps (rollup, etc.) require npm >= 8; the old
# `npm@7.24.0` pin here made `npm ci` extremely slow / hang on a cold cache.

# python is a dependency of node-gyp that we need to build node-sass https://www.npmjs.com/package/node-gyp
#RUN apk add g++ make python3 && \
#    npm config set python "$(which python3)"

#RUN npm install
COPY docker/init/ui-dev /init

EXPOSE 80
ENTRYPOINT ["/init"]

#FROM node:10.11-alpine AS ui-builder
FROM ui-dev AS ui-builder

#npm ci # installs what is specified in package-lock.json
RUN npm ci --no-save
#COPY ui/yarn.lock /app/

#RUN yarn install --frozen-lockfile

#RUN npm test
RUN npm run build


#
# PROD
FROM alpine:3.14.1 AS prod

ARG BUILD_VERSION
ENV SENTRY_RELEASE=$BUILD_VERSION

RUN set -eux; \
	apk add --update --no-cache \
	    bash \
	    coreutils \
	    libc6-compat \
	    ca-certificates \
        && \
	rm -rf /tmp/* /var/cache/apk/*

WORKDIR /app
COPY --from=build /app/api /tilmelding-api
COPY --from=ui-builder /app/dist /www
COPY docker/bin/init /init

#HEALTHCHECK --interval=30s --timeout=15s --start-period=900s --retries=3 CMD test -f /tmp/healthy
EXPOSE 80
ENTRYPOINT ["/init"]
CMD ["/tilmelding-api"]
