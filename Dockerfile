ARG VERSION
FROM mondoo/cnquery:$VERSION
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]