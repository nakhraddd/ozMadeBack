FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

# Generate templ components
RUN templ generate

RUN go build -o ./out/ozmade ./cmd/ozmade/main.go

EXPOSE 8080

CMD [ "/app/out/ozmade" ]
