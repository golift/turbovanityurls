#
# This is part of Application Builder.
# https://github.com/golift/application-builder
#

ARG ARCH=amd64
ARG OS=linux
ARG BUILD_DATE=0
ARG COMMIT=0
ARG VERSION=unknown
ARG BINARY=application-builder

FROM golang:latest as builder
ARG ARCH
ARG OS
ARG BINARY

RUN mkdir -p $GOPATH/pkg/mod $GOPATH/bin $GOPATH/src /${BINARY}
COPY . /${BINARY}
WORKDIR /${BINARY}

RUN apt update && \
    apt install -y upx cpio rpm2cpio && \
    CGO_ENABLED=0 make ${BINARY}.${ARCH}.${OS}

FROM scratch
ARG ARCH
ARG OS
ARG BUILD_DATE
ARG COMMIT
ARG VERSION
ARG LICENSE=MIT
ARG BINARY
ARG SOURCE_URL=http://github.com/golift/application-builder
ARG DESC=application-builder
ARG VENDOR=golift
ARG AUTHOR=golift
ARG CONFIG_FILE=config.conf

# Build-time metadata as defined at https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.title="${BINARY}" \
      org.opencontainers.image.documentation="${SOURCE_URL}/wiki/Docker" \
      org.opencontainers.image.description="${DESC}" \
      org.opencontainers.image.url="${SOURCE_URL}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.source="${SOURCE_URL}" \
      org.opencontainers.image.vendor="${VENDOR}" \
      org.opencontainers.image.authors="${AUTHOR}" \
      org.opencontainers.image.architecture="${OS} ${ARCH}" \
      org.opencontainers.image.licenses="${LICENSE}" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=builder /${BINARY}/${BINARY}.${ARCH}.${OS} /image

# Make sure we have an ssl cert chain and timezone data.
COPY --from=builder /etc/ssl /etc/ssl
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

ENV TZ=UTC

ENTRYPOINT [ "/image" ]
