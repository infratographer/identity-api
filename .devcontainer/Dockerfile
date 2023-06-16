# Used to install CRDB into the devcontainer
FROM cockroachdb/cockroach:latest-v22.2 as CRDB

FROM mcr.microsoft.com/vscode/devcontainers/go:1-1.20-bullseye

RUN mkdir /home/vscode/.ssh && \
    chown vscode:vscode /home/vscode/.ssh && \
    chmod 0700 /home/vscode/.ssh

# Set up crdb
RUN mkdir /usr/local/lib/cockroach
COPY --from=CRDB /cockroach/cockroach /usr/local/bin
COPY --from=CRDB /usr/local/lib/cockroach/libgeos.so /usr/local/lib/cockroach/
COPY --from=CRDB /usr/local/lib/cockroach/libgeos_c.so /usr/local/lib/cockroach/

# Install general use tooling
RUN export DEBIAN_FRONTEND=noninteractive \
    && curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg \
    &&  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | gpg --dearmor > /usr/share/keyrings/yarn-archive-keyring.gpg \
    && apt-get install apt-transport-https --yes \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list \
    && apt-get update \
    && apt-get -y install --no-install-recommends \
        bash-completion \
        helm \
        uuid-runtime

USER vscode

RUN go install -v github.com/cweill/gotests/gotests@v1.6.0

WORKDIR /workspace
