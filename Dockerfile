FROM gcr.io/distroless/static

ARG TARGETOS TARGETARCH

COPY kmfddm-$TARGETOS-$TARGETARCH /usr/bin/kmfddm

EXPOSE 9002

ENTRYPOINT ["/usr/bin/kmfddm"]
