FROM alpine
COPY static /static
COPY gotli /gotli
CMD ["/gotli"]

