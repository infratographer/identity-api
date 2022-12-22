FROM mcr.microsoft.com/vscode/devcontainers/go:0-1.19-bullseye

USER vscode

RUN mkdir /home/vscode/.ssh && \
    chown vscode:vscode /home/vscode/.ssh && \
    chmod 0700 /home/vscode/.ssh

WORKDIR /workspace
