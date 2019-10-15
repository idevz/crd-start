FROM golang
MAINTAINER idevz, zhoujing_k49@163.com

ADD entrypoint.sh /entrypoint.sh
ADD artifacts /artifacts
ADD crdstart /

ENTRYPOINT ["/entrypoint.sh", "/crdstart"]
