FROM scratch

COPY tmp/blackfish /blackfish

EXPOSE 9999

CMD ["/blackfish"]
