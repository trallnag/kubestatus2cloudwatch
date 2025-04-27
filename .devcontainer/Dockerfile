FROM  mcr.microsoft.com/devcontainers/base:latest@sha256:eb406d55b2ecd2f8a31d23e4bee50af112d688d169da7457f436f5825c3cef12

USER vscode

SHELL ["/usr/bin/bash", "-eux", "-o", "pipefail", "-c"]

# Set up general stuff.
RUN <<-EOT
  export DEBIAN_FRONTEND=noninteractive

  # Install basic tools.
  sudo apt-get update
  sudo apt-get install --yes --no-install-recommends \
    build-essential \
    curl \
    gnupg2

  # Add line break to included theme.
  sed --in-place "s/\$ /'\$'\\\n''❯ /g" \
    ~/.oh-my-zsh/custom/themes/devcontainers.zsh-theme
EOT

# Install Homebrew.
RUN <<-EOT
  export NONINTERACTIVE=1

  installer=https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
  curl --location --silent --show-error --fail $installer | bash
EOT

# Set up stuff related to Homebrew.
RUN <<-EOT
  eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

  # Install basic tools.
  brew install \
    jq \
    zsh-completions

  # Set up integration with Bash.
  > /tmp/bashrc
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/bashrc
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/bashrc
  cat ~/.bashrc >> /tmp/bashrc
  mv /tmp/bashrc ~/.bashrc

  # Set up integration with Zsh.
  > /tmp/zshrc
  echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /tmp/zshrc
  echo 'if [ -z "$LC_ALL" ]; then export LC_ALL=en_US.utf8; fi' >> /tmp/zshrc
  echo 'FPATH=/home/linuxbrew/.linuxbrew/share/zsh-completions:$FPATH' >> /tmp/zshrc
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

  tools=(
    copier
    exec-cmds-defer-errors
    filter-pre-commit-hooks
    mdformat
    pre-commit
  )

  for tool in ${tools[@]}; do
    uv tool install $tool
  done
EOT

CMD ["/usr/bin/zsh", "--login"]
