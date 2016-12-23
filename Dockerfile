FROM scratch

COPY tmp/smudge /smudge

EXPOSE 9999

CMD ["/smudge"]
