FROM golang:1.25.1-alpine

WORKDIR /app

# Install air for hot reloading
RUN go install github.com/air-verse/air@latest

# Add git for possible private dependencies
RUN apk add --no-cache git

ARG SERVICE_NAME
ENV SERVICE_NAME=${SERVICE_NAME}
ENV GOPROXY="direct"

# Command uses air to watch the /app directory (which we will bind mount)
CMD air --build.cmd "cd services/${SERVICE_NAME} && go build -v -o /tmp/main ./cmd/server" --build.bin "/tmp/main" --build.exclude_dir "templates,assets" --build.include_ext "go,tpl,tmpl,html,env" --build.stop_on_error "false"
