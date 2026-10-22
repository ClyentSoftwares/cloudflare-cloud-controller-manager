FROM golang:1.27 AS builder
WORKDIR /ccm
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=unspecified
RUN CGO_ENABLED=0 go build \
    -ldflags "-X 'github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cloudflare.providerVersion=${VERSION}'" \
    -o /cloudflare-cloud-controller-manager .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /cloudflare-cloud-controller-manager /bin/cloudflare-cloud-controller-manager
ENTRYPOINT ["/bin/cloudflare-cloud-controller-manager"]
