# Two stages. The build stage has the toolchain and nothing else; the run stage has
# no shell at all, which is why the fixtures and the verdict log are copied into the
# image rather than fetched at start: the container can produce a digest with no
# network and no token.
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/radar ./cmd/radar

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/radar /usr/local/bin/radar
COPY testdata /app/testdata
COPY decisions /app/decisions
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/radar"]
CMD ["--limit", "6"]
