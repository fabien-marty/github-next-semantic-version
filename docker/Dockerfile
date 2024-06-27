FROM alpine:3.20

RUN apk add git
COPY github-next-semantic-version /
COPY entrypoint.sh /
RUN mkdir /work
ENTRYPOINT ["/entrypoint.sh"]
