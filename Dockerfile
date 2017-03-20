FROM alpine:3.5
COPY static /static
COPY gotli /gotli
EXPOSE 8000
CMD ["/gotli"]

