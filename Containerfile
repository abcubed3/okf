# GoReleaser stages the platform-specific release binary as `okf` in this build context.
# Keep this file focused on packaging that binary; Go compilation happens in GoReleaser.
FROM gcr.io/distroless/static-debian13:nonroot

COPY okf /usr/local/bin/okf

WORKDIR /bundle

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/okf"]
