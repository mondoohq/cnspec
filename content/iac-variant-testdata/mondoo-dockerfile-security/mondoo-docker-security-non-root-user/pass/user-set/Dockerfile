FROM debian:12-slim
RUN groupadd -r appgroup && useradd -r -g appgroup appuser
USER appuser
CMD ["/app/server"]
