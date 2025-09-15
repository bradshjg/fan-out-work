# syntax=docker/dockerfile:1

FROM mcr.microsoft.com/devcontainers/go AS dev

USER vscode

RUN go install github.com/a-h/templ/cmd/templ@v0.3.943

FROM golang:latest AS build

WORKDIR /src

COPY src/fan-out-work/go.mod src/fan-out-work/go.sum ./
RUN go mod download

COPY src/fan-out-work/* ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /fan-out-work

FROM gcr.io/distroless/base-debian12 AS release

WORKDIR /

COPY --from=build /fan-out-work /fan-out-work

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/fan-out-work"]
