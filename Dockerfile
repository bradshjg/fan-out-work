# syntax=docker/dockerfile:1

FROM mcr.microsoft.com/devcontainers/go AS dev

USER vscode

RUN go install github.com/a-h/templ/cmd/templ@v0.3.943

RUN curl -s https://raw.githubusercontent.com/lindell/multi-gitter/master/install.sh | sh

FROM golang:latest AS build

WORKDIR /src

COPY src/fan-out-work/go.mod src/fan-out-work/go.sum .
RUN go mod download

COPY src/fan-out-work/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fan-out-work

FROM registry.access.redhat.com/ubi9/ubi AS release

WORKDIR /

COPY --from=dev /usr/local/bin/multi-gitter /usr/local/bin/multi-gitter
COPY --from=build /fan-out-work /fan-out-work
COPY --from=build /src/patches /patches

EXPOSE 8080

RUN chown -R 1001:0 /patches && chmod -R g=u /patches

USER 1001

ENTRYPOINT ["/fan-out-work"]
