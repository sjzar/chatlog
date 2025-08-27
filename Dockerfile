FROM debian:12-slim

LABEL maintainer="Sarv <https://github.com/sjzar>"

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata curl && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -r -g 1001 chatlog && \
    useradd -r -u 1001 -g chatlog -m -d /home/chatlog chatlog && \
    mkdir -p /app/data /app/work && \
    chown -R chatlog:chatlog /app

USER chatlog

WORKDIR /app

COPY --from=mwader/static-ffmpeg:7.1.1 --chown=chatlog:chatlog /ffmpeg /usr/local/bin/

COPY --chown=chatlog:chatlog chatlog /usr/local/bin/chatlog

EXPOSE 5030

ENV CHATLOG_DATA_DIR=/app/data \
    CHATLOG_WORK_DIR=/app/work \
    CHATLOG_HTTP_ADDR=0.0.0.0:5030 \
    PATH="/usr/local/bin:${PATH}"

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:5030/health || exit 1

CMD ["chatlog", "server"]