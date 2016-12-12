FROM scratch

COPY bin/blackfish /blackfish

EXPOSE 9999

CMD ["/blackfish"]
