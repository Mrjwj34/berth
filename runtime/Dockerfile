# Build once per toolchain, share image layers across all workspace containers.
# For Go projects: --build-arg BASE_IMAGE=golang:1.25-bookworm
ARG BASE_IMAGE=debian:bookworm-slim
FROM ${BASE_IMAGE}
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash ca-certificates curl git python3 socat \
    && rm -rf /var/lib/apt/lists/*
# Release artifact hashes are pinned, not fetched from an untrusted sibling URL.
RUN set -eu; arch="${TARGETARCH:-$(dpkg --print-architecture)}"; \
    case "$arch" in \
      amd64) hash=9b6dbc38324c0b0481f1cd1dd828ffdc78117129ec797678f4bf8c4023311281 ;; \
      arm64) hash=52fa7d5a2d5e0db470faec5976204fc215ed7e3d13689e930cf522becfb63778 ;; \
      *) echo "unsupported runtime image architecture: $arch" >&2; exit 1 ;; \
    esac; \
    curl --fail --location --retry 3 -o /tmp/pc.tar.gz \
      "https://github.com/F1bonacc1/process-compose/releases/download/v1.122.0/process-compose_linux_${arch}.tar.gz"; \
    echo "$hash  /tmp/pc.tar.gz" | sha256sum -c -; \
    mkdir /tmp/pc; tar -xzf /tmp/pc.tar.gz -C /tmp/pc; \
    install -m 0755 /tmp/pc/process-compose /usr/local/bin/process-compose; \
    rm -rf /tmp/pc /tmp/pc.tar.gz
WORKDIR /workspace
CMD ["sleep", "infinity"]
