FROM golang
MAINTAINER idevz, zhoujing_k49@163.com

ADD entrypoint.sh /entrypoint.sh
ADD artifacts /artifacts
ADD crdstart-debug /


EXPOSE 40000

ADD files/dlv /usr/local/bin/dlv
RUN chmod +x /usr/local/bin/dlv
ENTRYPOINT ["/usr/local/bin/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "exec", "/crdstart-debug", "--"]
