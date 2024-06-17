FROM golang:1.22.3 as builder
WORKDIR /ccm
ADD go.mod go.sum /ccm/
RUN go mod download
ADD . /ccm/
RUN ls -al

# Export the version
ARG VERSION=unspecified
RUN echo $VERSION > VERSION

RUN CGO_ENABLED=0 go build -ldflags "-X 'github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cloudflare.providerVersion=${VERSION}'" -o cloudflare-cloud-controller-manager.bin github.com/ClyentSoftwares/cloudflare-cloud-controller-manager

FROM alpine:3.20
RUN apk add --no-cache ca-certificates bash
COPY --from=builder /ccm/cloudflare-cloud-controller-manager.bin /bin/cloudflare-cloud-controller-manager
ENTRYPOINT ["/bin/cloudflare-cloud-controller-manager"]
