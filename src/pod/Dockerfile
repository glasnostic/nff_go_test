FROM alpine:3.9

RUN apk update && \
    apk add --no-cache iperf3 iproute2 bash ethtool curl netcat-openbsd python arping

COPY . .

ENTRYPOINT ["bash"]
EXPOSE 8080 3091
