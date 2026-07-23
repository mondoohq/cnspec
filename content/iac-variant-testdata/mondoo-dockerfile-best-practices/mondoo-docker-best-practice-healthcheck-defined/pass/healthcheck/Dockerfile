FROM nginx:1.27-alpine
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/health || exit 1
EXPOSE 8080
