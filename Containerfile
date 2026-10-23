# GoReleaser stages platform-specific binaries in ${TARGETPLATFORM}/okf
# Keep this file focused on packaging that binary; Go compilation happens in GoReleaser.
FROM gcr.io/distroless/static-debian13:nonroot

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/okf /usr/local/bin/okf

WORKDIR /bundle

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/okf"]
