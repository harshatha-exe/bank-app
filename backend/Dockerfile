FROM golang:1.26-bookworm AS dev
ENV GOTOOLCHAIN=auto
RUN useradd --create-home --shell /bin/bash appuser \
    && go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN chown -R appuser:appuser /app
RUN chown -R appuser:appuser /go
USER appuser
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]