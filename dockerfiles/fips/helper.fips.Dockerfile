ARG GO_VERSION=1.21

FROM go-fips:${GO_VERSION}

WORKDIR /build
COPY . /build/

ARG GOOS=linux
ARG GOARCH=amd64

RUN BASE_DIR="out/binaries/gitlab-runner-helper" && \
    make "${BASE_DIR}/gitlab-runner-helper-fips" GOOS=${GOOS} GOARCH=${GOARCH} && \
    ls "${BASE_DIR}"| grep gitlab-runner-helper| xargs -I '{}' mv "${BASE_DIR}/{}" /gitlab-runner-helper-fips

RUN /bin/false

ARG GIT_LFS_VERSION=3.4.1
RUN microdnf remove git-lfs
RUN wget -O git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm https://packagecloud.io/github/git-lfs/packages/el/8/git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm/download && rpm -i git-lfs-${GIT_LFS_VERSION}-1.el8.x86_64.rpm && git-lfs install && git-lfs version
