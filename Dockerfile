FROM ubuntu:16.04

#Begin: install prerequisites
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        curl \
        git \
        libcurl3-dev \
        libfreetype6-dev \
        libpng12-dev \
        libzmq3-dev \
        locate \
        pkg-config \
        rsync \
        software-properties-common \
        sudo \
        unzip \
        vim \
        wget \
        zip \
        zlib1g-dev \
        && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
#End: install prerequisites

#Begin: install golang
ENV GOLANG_VERSION 1.11
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_SHA256_CHECKSUM b3fcf280ff86558e0559e185b601c9eade0fd24c900b4c63cd14d1d38613e499
ENV GOPATH $HOME/go
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin
RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz && \
    echo "$GOLANG_SHA256_CHECKSUM golang.tar.gz" | sha256sum -c - && \
    sudo tar -C /usr/local -xzf golang.tar.gz && \
    rm golang.tar.gz && \
    mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
#End: install golang

#Begin: install delve
RUN go get github.com/derekparker/delve/cmd/dlv
#End: install delve

#Begin: install seqkit - for developers
#RUN go get -u github.com/cznic/sortutil
#RUN go get -u github.com/dustin/go-humanize
#RUN go get -u github.com/edsrzf/mmap-go
#RUN go get -u github.com/mattn/go-colorable
#RUN go get -u github.com/tatsushid/go-prettytable
#RUN go get -u github.com/mitchellh/go-homedir
#RUN go get -u github.com/spf13/cobra
#RUN go get -u github.com/shenwei356/breader
#RUN go get -u github.com/shenwei356/bwt/fmi
#RUN go get -u github.com/shenwei356/bio/seq
#RUN go get -u github.com/shenwei356/bio/seqio/fastx
#RUN go get -u github.com/shenwei356/util/byteutil
#RUN go get -u github.com/shenwei356/util/math
#RUN go get -u github.com/shenwei356/natsort
#RUN go get -u github.com/shenwei356/bwt/fmi
#RUN go get -u github.com/shenwei356/go-logging
#RUN go get -u github.com/shenwei356/xopen

#ADD seqkit /go/src/github.com/<developer>/seqkit/seqkit
#ADD tests /go/src/github.com/<developer>/seqkit/tests
#RUN go get -u github.com/<developer>/bio/seq
#End: install seqkit - for developers

#Begin: install seqkit - for users
RUN go get -u github.com/shenwei356/seqkit/seqkit
#End: install seqkit - for users

#WORKDIR /go/src/github.com/<developer>/seqkit
WORKDIR /go/src/github.com/shenwei356/seqkit
