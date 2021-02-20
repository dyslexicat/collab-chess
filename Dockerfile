FROM golang:latest as builder
WORKDIR /chessbot
COPY go.mod go.sum ./
RUN go mod download
RUN go get github.com/dyslexicat/collab-chess/...
COPY . .
RUN go build -o main main.go

FROM ubuntu:latest
WORKDIR /app

RUN apt-get update && apt-get dist-upgrade -y && apt clean all
RUN apt-get update && apt-get install -y curl wget git clang-6.0 ninja-build protobuf-compiler libprotobuf-dev build-essential && apt-get clean all

RUN git clone https://github.com/official-stockfish/Stockfish.git &&\
    cd Stockfish &&\
    cd src &&\
    make help &&\
    make net &&\
    make build ARCH=x86-64-modern

COPY --from=builder /chessbot/main .
COPY --from=builder /chessbot/.env .
COPY --from=builder /chessbot/assets ./assets

EXPOSE 5000

CMD ["./main"]
