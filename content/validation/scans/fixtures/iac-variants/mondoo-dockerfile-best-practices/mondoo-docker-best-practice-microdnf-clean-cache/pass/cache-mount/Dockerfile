FROM redhat/ubi9-micro:9.4
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    microdnf install -y httpd
