FROM scratch
COPY static /static
COPY gotli /gotli
CMD ["/gotli"]

