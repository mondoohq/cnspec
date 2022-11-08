FROM mondoo/cnquery:7.3.0
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]