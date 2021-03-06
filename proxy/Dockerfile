# Proxy build and runtime
#
# Builds a slim runtime image with the conduit-proxy binary.

## Build the rust proxy into a binary.
#
# If the RELEASE arg is set and non-empty, a release artifact is built.
FROM gcr.io/runconduit/proxy-deps:b84ed5d0 as build
WORKDIR /usr/src/conduit
# Ranked roughly from least to most likely to change. Cargo.lock is the least likely
# because it is supposed to be cached in the deps base image.
COPY futures-mpsc-lossy ./futures-mpsc-lossy
COPY tower-router ./tower-router
COPY proto ./proto
COPY proxy ./proxy
ARG RELEASE
RUN if [ -z "$RELEASE" ]; \
    then cargo build --frozen -p conduit-proxy           && mv target/debug/conduit-proxy   target/conduit-proxy ; \
    else cargo build --frozen -p conduit-proxy --release && mv target/release/conduit-proxy target/conduit-proxy ; \
    fi

## Install the proxy binary into the base runtime image.
FROM gcr.io/runconduit/base:2017-10-30.01
COPY --from=build /usr/src/conduit/target/conduit-proxy /usr/local/bin/conduit-proxy
ENV CONDUIT_PROXY_LOG=info
ENTRYPOINT ["/usr/local/bin/conduit-proxy"]
