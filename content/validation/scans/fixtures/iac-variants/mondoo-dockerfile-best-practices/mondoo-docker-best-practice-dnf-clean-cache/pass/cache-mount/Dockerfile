FROM fedora:40
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    dnf install -y httpd
