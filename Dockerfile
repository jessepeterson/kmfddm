FROM gcr.io/distroless/static

ARG TARGETOS TARGETARCH

COPY kmfddm-$TARGETOS-$TARGETARCH /app/kmfddm

EXPOSE 9002

VOLUME ["/app/db"]

WORKDIR /app

ENTRYPOINT ["/app/kmfddm"]
