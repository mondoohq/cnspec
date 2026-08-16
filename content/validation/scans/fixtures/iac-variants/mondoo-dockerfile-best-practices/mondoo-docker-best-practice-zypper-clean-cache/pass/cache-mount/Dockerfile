FROM opensuse/leap:15.6
RUN --mount=type=cache,target=/var/cache/zypp,sharing=locked \
    zypper --non-interactive install httpd
