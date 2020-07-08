# Generate a docker for go-ts-segmenter
# by Jordi Cenzano
# VERSION               1.0.0

FROM golang:1.14
LABEL maintainer "Jordi Cenzano <jordi.cenzano@gmail.com>"

# Set workdir
WORKDIR /go/src/go-ts-segmenter

# Copy code
COPY . .

# Override Makefile 
COPY MakefileDocker Makefile

# Compile
RUN make

# Start
ENTRYPOINT ["bin/go-ts-segmenter"]
CMD ["-h"]