FROM debian:12-slim
RUN apt-get update --allow-insecure-repositories && \
    apt-get install -y --no-install-recommends somepackage && \
    rm -rf /var/lib/apt/lists/*
