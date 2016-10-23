FROM golang:1.7.1-alpine

# install curl 
RUN apk add --update curl && rm -rf /var/cache/apk/*

# copy deps
ADD vendor /go/src/
ADD xconsul /go/src/github.com/stefanprodan/xmicro/xconsul

# copy sources
RUN mkdir /xmicro 
ADD . /xmicro/ 

# build
WORKDIR /xmicro/app/
RUN go build -o xmicro . 

HEALTHCHECK CMD curl --fail http://localhost:8000/ping || exit 1

EXPOSE 8000/tcp

# run
CMD ["/xmicro/app/xmicro"]
