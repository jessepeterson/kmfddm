FROM gcr.io/distroless/static

COPY kmfddm-linux-amd64 /kmfddm

EXPOSE 9002

ENTRYPOINT ["/kmfddm"]
