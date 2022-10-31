FROM mondoo/cnquery:7.2.0
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]