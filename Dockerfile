# iron/go:dev is the alpine image with the go tools added
FROM iron/go:dev
WORKDIR /app
# Set an env var that matches your github repo name, replace treeder/dockergo here with your repo name
ENV SRC_DIR=/go/src/github.com/moos3/linkFN/
# Add the source code:
ADD . $SRC_DIR
RUN go get -u gopkg.in/mailgun/mailgun-go.v1
# Build it:
RUN cd $SRC_DIR; go build -o linkFn; cp linkFn /app/
EXPOSE 3000
ENTRYPOINT ["./linkFn"]
