ARG GO_VERSION=1.21
ARG BUILD_IMAGE=go-fips

FROM ${BUILD_IMAGE}:${GO_VERSION}

WORKDIR /build
COPY . /build/

ARG GOOS=linux
ARG GOARCH=amd64

RUN make runner-bin-fips GOOS=${GOOS} GOARCH=${GOARCH} && \
    cp out/binaries/* /

ARG GIT_LFS_VERSION=3.4.1
RUN microdnf remove git-lfs
RUN wget -O git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm https://packagecloud.io/github/git-lfs/packages/el/8/git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm/download && rpm -i git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm && git-lfs install && git-lfs version
