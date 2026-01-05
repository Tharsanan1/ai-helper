# AI Helper Sandbox Image
# This image provides a sandboxed environment for running AI coding CLIs
# with auto-accept enabled safely.

FROM ubuntu:24.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install common dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    openssh-client \
    ca-certificates \
    gnupg \
    python3 \
    python3-pip \
    python3-venv \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20.x (required for Claude CLI)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI
RUN npm install -g @anthropic-ai/claude-code

# Install GitHub Copilot CLI
RUN npm install -g @github/copilot

# Install Gemini CLI
RUN npm install -g @google/gemini-cli 2>/dev/null || \
    echo "Note: Gemini CLI installation may require manual setup"

# Create non-root user (required - Claude CLI doesn't allow --dangerously-skip-permissions as root)
RUN useradd -m -s /bin/bash developer && \
    mkdir -p /home/developer/.claude /home/developer/.copilot /home/developer/.gemini /home/developer/.ssh && \
    chown -R developer:developer /home/developer

# Switch to non-root user
USER developer

# Set up git to allow operations in mounted directories
RUN git config --global --add safe.directory '*'

# Set environment
ENV HOME=/home/developer
ENV PATH="/home/developer/.local/bin:$PATH"

# Set workspace directory
WORKDIR /workspace

# Default command
CMD ["/bin/bash"]
