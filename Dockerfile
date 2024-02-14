# Docker builder for Golang
FROM golang:latest as builder
# Install the Certificate-Authority certificates for the app to be able to make
# calls to HTTPS endpoints.
# Git is required for fetching the dependencies.
RUN apt install -y ca-certificates git

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./go.mod ./go.sum ./
RUN go mod download

# Import the code from the context.
COPY ./ ./

# Build binary
RUN go build -o bot . 

FROM golang:latest as app
RUN apt update && apt install ffmpeg -y 
RUN wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux && chmod 775 yt-dlp_linux && mv yt-dlp_linux /bin/yt-dlp

COPY --from=builder /src/bot /app/
WORKDIR /app

# Run the compiled binary.
ENTRYPOINT [ "./bot" ]