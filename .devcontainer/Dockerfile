FROM  mcr.microsoft.com/devcontainers/base:latest

USER vscode

SHELL ["/usr/bin/bash", "-eux", "-o", "pipefail", "-c"]

#
# Perform general setup.
# Install packages with APT.
#

RUN <<-EOT
  export DEBIAN_FRONTEND=noninteractive

  sudo apt-get update
  sudo apt-get install --yes --no-install-recommends \
    build-essential \
    curl \
    gnupg2
EOT

#
# Install Homebrew.
#

RUN <<-EOT
  export NONINTERACTIVE=1

  installer=https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
  curl --location --silent --show-error --fail $installer | bash
EOT

#
# Install general formulae.
# Set up Bash environment.
# Set up Zsh environment.
#

RUN <<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
  brew install zsh-completions

  > /tmp/bashrc
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/bashrc
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/bashrc
  cat ~/.bashrc >> /tmp/bashrc
  mv /tmp/bashrc ~/.bashrc

  > /tmp/zshrc
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/zshrc
  echo 'FPATH=/home/linuxbrew/.linuxbrew/share/zsh-completions:$FPATH' >> /tmp/zshrc
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/zshrc
  cat ~/.zshrc >> /tmp/zshrc
  mv /tmp/zshrc ~/.zshrc
EOT

# Install tools with Homebrew.
RUN <<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

	brew install \
    go \
    gofumpt \
    golangci-lint \
    goreleaser \
    just \
    shellcheck \
    shfmt \
    uv \
    yamlfmt
EOT

# Install tools with uv.
RUN <<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

	uv tool install copier
  uv tool install exec-cmds-defer-errors
  uv tool install filter-pre-commit-hooks
  uv tool install mdformat
  uv tool install pre-commit
EOT

CMD ["/usr/bin/zsh", "--login"]
