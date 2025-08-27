FROM debian:12-slim

LABEL maintainer="Sarv <https://github.com/sjzar>"

ARG DEBIAN_FRONTEND=noninteractive

ENV PUID=1000 PGID=1000
ENV GOSU_VERSION=1.17
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates tzdata curl wget; \
    wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$(dpkg --print-architecture)"; \
	chmod +x /usr/local/bin/gosu; \
	gosu --version; \
    groupadd -r -g ${PGID} chatlog; \
    useradd -r -u ${PUID} -g chatlog -m -d /home/chatlog chatlog; \
    mkdir -p /app/data /app/work; \
    apt-get purge -y --auto-remove wget; \
	rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY script/docker-entrypoint.sh /usr/local/bin/entrypoint.sh

COPY --from=mwader/static-ffmpeg:7.1.1 /ffmpeg /usr/local/bin/

COPY chatlog /usr/local/bin/chatlog

RUN chmod +x /usr/local/bin/entrypoint.sh \
             /usr/local/bin/ffmpeg \
             /usr/local/bin/chatlog

EXPOSE 5030

ENV CHATLOG_DATA_DIR=/app/data \
    CHATLOG_WORK_DIR=/app/work \
    CHATLOG_HTTP_ADDR=0.0.0.0:5030 \
    PATH="/usr/local/bin:${PATH}"

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:5030/health || exit 1

ENTRYPOINT ["entrypoint.sh"]

CMD ["chatlog", "server"]