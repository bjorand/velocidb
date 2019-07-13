FROM velocidb/pre_server

COPY . /go/src/github.com/bjorand/velocidb

RUN make install

EXPOSE 4300 4301

CMD velocidb
