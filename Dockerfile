FROM figassis/ubuntu-golang

COPY . /go/src/github.com/figassis/ispmon
RUN go get -u github.com/kardianos/govendor && go install github.com/kardianos/govendor
RUN cd /go/src/github.com/figassis/ispmon && govendor add +e && govendor sync && govendor fetch +m && go install
RUN mkdir -p /etc/ispmon && mv /go/bin/ispmon /etc/ispmon/
RUN mv /etc/ispmon/config.example.json /etc/ispmon/config.json && mv /go/src/github.com/figassis/ispmon/*.json /etc/ispmon/

ENTRYPOINT ["/etc/ispmon"]