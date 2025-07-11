# syntax=docker/dockerfile:1@sha256:9857836c9ee4268391bb5b09f9f157f3c91bb15821bb77969642813b0d00518d

FROM  mcr.microsoft.com/devcontainers/base:bookworm@sha256:ce2e9e611939e611b737362c045bb6d3449bb3efb84898525d724aace1737b90

USER vscode

SHELL ["/usr/bin/bash", "-eux", "-o", "pipefail", "-c"]

ARG APT_PKGS="\
  build-essential \
  curl \
  dnsutils \
  gnupg2 \
  python-is-python3 \
  python3 \
  xxd \
"

ARG BREW_PKGS="\
  cosign \
  gh \
  go \
  jq \
  just \
  mise \
  uv \
  watchexec \
  zsh-completions \
"

# Install various packages with APT.
RUN <<-EOT
  export DEBIAN_FRONTEND=noninteractive
  sudo apt-get update
  sudo apt-get install --yes --no-install-recommends $APT_PKGS
EOT

# Install Homebrew using the official installer.
RUN <<-EOT
  export NONINTERACTIVE=1
  installer=https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
  curl --location --silent --show-error --fail $installer | bash
EOT

# Install various packages with Homebrew.
RUN \
  --mount=type=secret,id=github-token,env=GITHUB_TOKEN \
  --mount=type=secret,id=homebrew-github-api-token,env=HOMEBREW_GITHUB_API_TOKEN \
<<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
  brew install $BREW_PKGS
EOT

# Configure various stuff.
RUN <<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

  # Add line break to included theme. Would be nice to make it colorized, but so
  # far I haven't found a way to do that. It always breaks one way or another.
  sed --in-place "s/\$ /'\$'\\\n''❯ /g" \
    ~/.oh-my-zsh/custom/themes/devcontainers.zsh-theme

  #
  # Set up Homebrew integration with Bash.
  #

  # Set up temporary file.
  > /tmp/bashrc

  # Environment.
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/bashrc

  # Locale.
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/bashrc

  # Mise.
  echo 'eval "$(mise activate bash --shims)"' >> /tmp/zshrc
  echo 'eval "$(mise activate bash)"' >> /tmp/bashrc

  # Combine with existing file.
  cat ~/.bashrc >> /tmp/bashrc
  mv /tmp/bashrc ~/.bashrc

  #
  # Set up Homebrew integration with Zsh.
  #

  # Set up temporary file.
  > /tmp/zshrc

  # Environment.
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/zshrc

  # Locale.
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/zshrc

  # Completions.
  echo 'FPATH=/home/linuxbrew/.linuxbrew/share/zsh-completions:$FPATH' >> /tmp/zshrc

  # Mise.
  echo 'eval "$(mise activate zsh --shims)"' >> /tmp/zshrc
  echo 'eval "$(mise activate zsh)"' >> /tmp/zshrc

  # Combine with existing file.
  cat ~/.zshrc >> /tmp/zshrc
  mv /tmp/zshrc ~/.zshrc
EOT

CMD ["/usr/bin/env", "zsh", "--interactive" , "--login"]
