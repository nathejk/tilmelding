#
# DEV - BASE
FROM golang:1.22 AS api-dev

RUN mkdir -p /app/docker/bin
WORKDIR /app

ENV DEBIAN_FRONTEND noninteractive
RUN set -eux; \
	apt -qq update && \
    apt install -qq -y --no-install-recommends \
        bash \
        inotify-tools && \
    apt -qq clean

RUN go install honnef.co/go/tools/cmd/staticcheck@latest

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

# test and lint before building
RUN go test -timeout 60s ./... && \
    staticcheck ./...

# build service
RUN GOARCH=amd64 CGO_ENABLED=1 GOOS=linux \
	go build -a -ldflags '-extldflags "-static"' -ldflags="-w -s" -o api nathejk.dk/cmd/api

## UI
FROM node:20.11.1-alpine3.19 AS ui-dev

RUN mkdir -p /app
WORKDIR /app
COPY vue /app

RUN npm install -g npm@7.24.0

# python is a dependency of node-gyp that we need to build node-sass https://www.npmjs.com/package/node-gyp
#RUN apk add g++ make python3 && \
#    npm config set python "$(which python3)"

#RUN npm install
COPY docker/init/ui-dev /init

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

ARG CI_VERSION
ENV SENTRY_RELEASE=$CI_VERSION

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
