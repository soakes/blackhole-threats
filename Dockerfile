# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-trixie AS build

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w -X github.com/netspeedy/blackhole-threats/internal/buildinfo.Version=${VERSION} -X github.com/netspeedy/blackhole-threats/internal/buildinfo.Commit=${VCS_REF} -X github.com/netspeedy/blackhole-threats/internal/buildinfo.BuildDate=${BUILD_DATE}" \
      -o /out/blackhole-threats \
      ./cmd/blackhole-threats

FROM debian:trixie-slim

ARG TARGETPLATFORM
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown
ARG S6_OVERLAY_VERSION=3.2.3.0
ARG S6_OVERLAY_NOARCH_SHA256=b720f9d9340efc8bb07528b9743813c836e4b02f8693d90241f047998b4c53cf
ARG S6_OVERLAY_X86_64_SHA256=a93f02882c6ed46b21e7adb5c0add86154f01236c93cd82c7d682722e8840563
ARG S6_OVERLAY_AARCH64_SHA256=0952056ff913482163cc30e35b2e944b507ba1025d78f5becbb89367bf344581
ARG S6_OVERLAY_SYMLINKS_NOARCH_SHA256=a60dc5235de3ecbcf874b9c1f18d73263ab99b289b9329aa950e8729c4789f0e
ARG S6_OVERLAY_SYMLINKS_ARCH_SHA256=8e8407a0cf84fd8884cc7de52049bdce71bf3993f07cbaa2a7c8cebc9581d805

# Set environment variable to minimize s6 output
ENV S6_LOGGING=0 \
    S6_VERBOSITY=0

LABEL org.opencontainers.image.title="blackhole-threats" \
      org.opencontainers.image.description="BGP blackhole route server for threat feeds" \
      org.opencontainers.image.url="https://github.com/netspeedy/blackhole-threats" \
      org.opencontainers.image.source="https://github.com/netspeedy/blackhole-threats" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.licenses="MIT"

# Install required packages
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
         ca-certificates \
         tzdata \
         xz-utils \
         wget

# Install the pinned s6-overlay release. Automation keeps this current.
RUN set -eux \
    && case ${TARGETPLATFORM} in \
         "linux/amd64") \
           S6_OVERLAY_ARCH="x86_64" ; \
           S6_OVERLAY_ARCH_SHA256="${S6_OVERLAY_X86_64_SHA256}" \
           ;; \
         "linux/arm64") \
           S6_OVERLAY_ARCH="aarch64" ; \
           S6_OVERLAY_ARCH_SHA256="${S6_OVERLAY_AARCH64_SHA256}" \
           ;; \
         *) \
           echo "Unsupported TARGETPLATFORM: ${TARGETPLATFORM}" >&2 ; \
           exit 1 \
           ;; \
    esac \
    && echo "Installing S6 Overlay version: ${S6_OVERLAY_VERSION}" \
    && cd /tmp \
    && wget -q https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz \
    && echo "${S6_OVERLAY_NOARCH_SHA256}  s6-overlay-noarch.tar.xz" | sha256sum -c - \
    && tar -Jxpf s6-overlay-noarch.tar.xz -C / \
    && wget -q https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-${S6_OVERLAY_ARCH}.tar.xz \
    && echo "${S6_OVERLAY_ARCH_SHA256}  s6-overlay-${S6_OVERLAY_ARCH}.tar.xz" | sha256sum -c - \
    && tar -Jxpf s6-overlay-${S6_OVERLAY_ARCH}.tar.xz -C / \
    && wget -q https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-symlinks-noarch.tar.xz \
    && echo "${S6_OVERLAY_SYMLINKS_NOARCH_SHA256}  s6-overlay-symlinks-noarch.tar.xz" | sha256sum -c - \
    && tar -Jxpf s6-overlay-symlinks-noarch.tar.xz -C / \
    && wget -q https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-symlinks-arch.tar.xz \
    && echo "${S6_OVERLAY_SYMLINKS_ARCH_SHA256}  s6-overlay-symlinks-arch.tar.xz" | sha256sum -c - \
    && tar -Jxpf s6-overlay-symlinks-arch.tar.xz -C / \
    && rm -f /tmp/s6-overlay-*.tar.xz \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/blackhole-threats /usr/sbin/blackhole-threats
COPY packaging/container/rootfs/ /

# Create required directories.
RUN set -x \
    && mkdir -p \
         /config \
         /var/log/blackhole-threats

# Set permissions
RUN chmod +x \
    /usr/local/bin/blackhole-threats-entrypoint \
    /etc/s6-overlay/s6-rc.d/blackhole-threats/run \
    /etc/s6-overlay/s6-rc.d/blackhole-threats/log/run \
    /etc/s6-overlay/s6-rc.d/blackhole-threats/finish

EXPOSE 179

ENTRYPOINT ["/init"]
CMD ["/usr/local/bin/blackhole-threats-entrypoint"]
