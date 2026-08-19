# Keep the complete Go toolchain so code can be edited, built, and tested in the container.
FROM golang:1.22

WORKDIR /app

ENV PATH="/go/bin:/usr/local/go/bin:${PATH}"
RUN printf '%s\n' 'export PATH=/go/bin:/usr/local/go/bin:$PATH' > /etc/profile.d/go-path.sh

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build ./...

CMD ["bash"]
