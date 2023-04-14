FROM gcr.io/distroless/static

# Copy the binary that goreleaser built
COPY identity-api /identity-api

# Run the web service on container startup.
ENTRYPOINT ["/identity-api"]
CMD ["serve"]
