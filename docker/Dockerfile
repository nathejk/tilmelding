#
# DEV - BASE
FROM golang:1.17 AS dev

RUN mkdir -p /app/docker/bin
WORKDIR /app

ENV DEBIAN_FRONTEND noninteractive
RUN set -eux; \
	apt -qq update && \
    apt install -qq -y --no-install-recommends \
        bash \
        inotify-tools && \
    apt -qq clean

RUN printf "#!/bin/sh\necho 'Please mount repo into /app'" > /app/docker/bin/init-dev && \
    chmod +x /app/docker/bin/init-dev

ENTRYPOINT ["/app/docker/bin/init-dev", "api"]


#
# BASE
FROM dev AS base

# copy and download go mod dependencies
COPY go/go.mod go/go.sum ./
RUN go mod download

# copy src
COPY go/ .

#
# BUILD
FROM base AS build

# build service
RUN GOARCH=amd64 CGO_ENABLED=1 GOOS=linux \
	go build -a -ldflags="-w -s" -o api nathejk.dk/cmd/api

## UI
FROM node:15.14-alpine3.13 AS ui-dev

RUN mkdir -p /app
WORKDIR /app
COPY js /app

RUN npm install -g npm@7.24.0

# python is a dependency of node-gyp that we need to build node-sass https://www.npmjs.com/package/node-gyp
#RUN apk add g++ make python3 && \
#    npm config set python "$(which python3)"

#RUN npm install

ENTRYPOINT ["/app/init-dev"]

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

ENTRYPOINT ["/init"]
CMD ["/tilmelding-api"]
