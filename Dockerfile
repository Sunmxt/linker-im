FROM golang:1.11.5-alpine3.7

COPY ./ /build/
ARG MAKE_ENV

RUN set -xe;\
    sed -Ei "s/dl-cdn\.alpinelinux\.org/mirrors.tuna.tsinghua.edu.cn/g" /etc/apk/repositories;\
    mkdir /apk-cache;\
    apk update --cache-dir /apk-cache;\
    apk add -t build-deps gcc make g++ git;\
    cd /build;\
    make $MAKE_ENV;\
    cp bin/linker-gate /bin/;\
    cp bin/linker-svc /bin/;\
    apk del build-deps;\
    rm -rf /build /apk-cache /root/.cache;

CMD ["linker-svc"]
