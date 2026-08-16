FROM centos:7
RUN --mount=type=cache,target=/var/cache/yum,sharing=locked \
    yum install -y httpd
