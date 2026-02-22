# cosmic-deb

`cosmic-deb` is a build orchestration tool, written in Go, that automates the retrieval, compilation, and packaging of the COSMIC Desktop Environment from the upstream [pop-os](https://github.com/pop-os) organisation into installable Debian binary packages (`.deb`). The tool is designed to operate both within containerised environments and directly upon host systems, provided the underlying platform is APT-based and satisfies the minimum distribution version requirements outlined below.

The project additionally provides a suite of Bash shell scripts to facilitate the installation and removal of the produced packages, whether sourced from a local build or downloaded directly from published GitHub releases.

---

## System Requirements

The following distributions are supported. For Ubuntu, only LTS (Long Term Support) and current development (devel) releases are supported. Non-LTS intermediate releases are explicitly unsupported and will cause the tool to terminate with an error.

| Distribution | Supported Releases |
|--------------|-----------------------|
| Debian       | 12 (bookworm) or later |
| Ubuntu       | LTS and devel (e.g., jammy, noble, or plucky) |

Distributions not derived from Debian or Ubuntu are explicitly unsupported. The tool enforces APT availability at startup and will not proceed on incompatible systems.

---

## Building the `cosmic-deb` Binary

Go 1.21 or later is required. To compile the binary from source:

```bash
git clone https://github.com/jimed-rand/cosmic-deb.git
cd cosmic-deb
make build
```

Alternatively, using the Go toolchain directly:

```bash
go build -trimpath -ldflags "-s -w" -o cosmic-deb .
```

---

## Repository Configuration (`repos.json`)

All repository metadata — names, upstream URLs, and per-repo epoch tags — is stored in `repos.json` at the root of the project. This file is the single source of truth for which components are built and at what version.

```json
{
  "generated_at": "2026-02-22",
  "epoch_latest": "epoch-1.0.7",
  "repos": [
    {
      "name": "cosmic-term",
      "url": "https://github.com/pop-os/cosmic-term",
      "tag": "epoch-1.0.7"
    },
    {
      "name": "cosmic-theme-extra",
      "url": "https://github.com/pop-os/cosmic-theme-extra",
      "tag": "",
      "branch": "master"
    }
  ]
}
```

Each entry specifies a `tag` to use when downloading the source tarball. Repositories that carry no epoch tags (such as `cosmic-theme-extra`) specify a `branch` instead; the tool downloads the branch HEAD tarball in that case.

### Updating `repos.json` Automatically

To re-fetch the latest epoch tag for every repository directly from GitHub and rewrite `repos.json`:

```bash
./cosmic-deb -update-repos
```

This uses `git ls-remote --tags` (no API token required, not subject to rate limiting) and updates only the `tag` and `epoch_latest` fields, preserving all other configuration. Run this command prior to a new build cycle whenever upstream epoch releases are published.

---

## Usage

### Building All COSMIC Components

The following command initiates the full build pipeline. Root privileges are required to invoke `apt-get` for dependency installation.

```bash
sudo ./cosmic-deb
```

By default, each component is downloaded at the tag specified in `repos.json`. To override all repos to a specific tag globally:

```bash
sudo ./cosmic-deb -tag epoch-1.0.7
```

Alternatively, via the provided Makefile target:

```bash
make run
make run TAG=epoch-1.0.7
```

### Meta Package

Upon successful completion of a full build (i.e. without `-only`), a `cosmic-desktop` meta package is produced in addition to the individual component packages. This meta package declares dependencies on all successfully built COSMIC components and their runtime libraries, enabling a single `dpkg -i` invocation to install the entire desktop environment:

```bash
sudo dpkg -i cosmic-packages/cosmic-desktop_*.deb
sudo apt-get install -f
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-repos` | `repos.json` | Path to the repository configuration file |
| `-update-repos` | `false` | Fetch latest epoch tags from upstream, rewrite the repos config, then exit |
| `-tag` | _(empty)_ | Override tag for all repos. When omitted, per-repo tags from `repos.json` are used. |
| `-workdir` | `cosmic-work` | Working directory for source checkout and compilation |
| `-outdir` | `cosmic-packages` | Output directory for the produced `.deb` packages |
| `-jobs` | CPU count | Number of parallel compilation jobs |
| `-skip-deps` | `false` | Skip automatic installation of build dependencies |
| `-only` | _(empty)_ | Restrict the build to a single named component matching a `name` in `repos.json` |
| `-tui` | `false` | Launch the interactive TUI wizard and build monitor |

### Usage Examples

```bash
sudo ./cosmic-deb -tag epoch-1.0.7 -jobs 8 -outdir /tmp/debs

sudo ./cosmic-deb -only cosmic-term -skip-deps

sudo ./cosmic-deb -repos /etc/cosmic-deb/repos.json -workdir /mnt/build

./cosmic-deb -update-repos
```

---

## How Source Archives Are Fetched

For each component with a `tag` entry in `repos.json`, the source archive is downloaded as a `.tar.gz` from GitHub:

```
https://github.com/pop-os/<component>/archive/refs/tags/<tag>.tar.gz
```

For components with no epoch tag (e.g. `cosmic-theme-extra`), the branch HEAD is used instead:

```
https://github.com/pop-os/<component>/archive/refs/heads/<branch>.tar.gz
```

The tool uses `tar -tzf` to inspect the archive and determine the exact top-level directory name before extraction, eliminating guesswork around GitHub's naming convention. The archive is removed after successful extraction.

When running `-update-repos`, tags are retrieved via `git ls-remote --tags` which requires no authentication token and is not subject to GitHub API rate limiting.

---

## Installation

### From a Local Build

To install the entire COSMIC desktop environment using the generated meta package:

```bash
sudo dpkg -i cosmic-packages/cosmic-desktop_*.deb
sudo apt-get install -f
```

Alternatively, to install all packages directly without the meta package:

```bash
sudo bash scripts/install-local.sh cosmic-packages
```

A custom output directory may be specified as an argument:

```bash
sudo bash scripts/install-local.sh /path/to/output/directory
```

### From a Published GitHub Release

To download and install packages from the most recent published release:

```bash
curl -fsSL https://raw.githubusercontent.com/jimed-rand/cosmic-deb/main/scripts/install-release.sh | sudo bash
```

To target a specific release tag:

```bash
curl -fsSL https://raw.githubusercontent.com/jimed-rand/cosmic-deb/main/scripts/install-release.sh | sudo bash -s -- v1.0.0
```

---

## Uninstallation

To remove all COSMIC components that were installed by this tool:

```bash
sudo bash scripts/uninstall.sh
```

The script will enumerate all recognised COSMIC packages present on the system, present a confirmation prompt, and proceed to remove them via `apt-get remove` followed by `apt-get autoremove`.

---

## Makefile Targets

The following targets are available via `make`:

| Target | Description |
|--------|-------------|
| `build` | Compile the `cosmic-deb` binary |
| `clean` | Remove the compiled binary and output package directory |
| `install` | Install the binary and scripts to `PREFIX` (default: `/usr/local`) |
| `uninstall` | Remove the installed binary and scripts |
| `run` | Compile and execute the full build pipeline |
| `run-skip-deps` | Execute the build pipeline without installing dependencies |
| `run-only` | Build a single component (requires `COMPONENT=<n>`) |
| `update-repos` | Re-fetch latest epoch tags and update `repos.json` |
| `fmt` | Format Go source files using `go fmt` |
| `vet` | Run static analysis using `go vet` |
| `tidy` | Tidy the Go module dependency graph |
| `help` | Display usage information for all targets |

### Makefile Variables

```bash
make run TAG=epoch-1.0.7 JOBS=4 OUTDIR=/tmp/debs

make run-only COMPONENT=cosmic-term

make install PREFIX=/usr DESTDIR=/tmp/staging
```

---

## Container Support

`cosmic-deb` is fully compatible with containerised build environments, including Docker, LXC, and `systemd-nspawn`, provided the container image is based on a supported Debian or Ubuntu release.

**Example — building within a Docker container:**

```bash
docker run --rm -it \
  -v "$(pwd)/output":/output \
  debian:bookworm bash -c \
  "apt-get update && apt-get install -y golang make git && \
   git clone https://github.com/jimed-rand/cosmic-deb.git /src && \
   cd /src && make run TAG=epoch-1.0.7 OUTDIR=/output"
```

---

## Components Built

The following upstream repositories are included in `repos.json` and processed by the build pipeline:

| Component | Source | Tag Strategy |
|-----------|--------|--------------|
| `cosmic-applets` | [pop-os/cosmic-applets](https://github.com/pop-os/cosmic-applets) | epoch tag |
| `cosmic-applibrary` | [pop-os/cosmic-applibrary](https://github.com/pop-os/cosmic-applibrary) | epoch tag |
| `cosmic-bg` | [pop-os/cosmic-bg](https://github.com/pop-os/cosmic-bg) | epoch tag |
| `cosmic-comp` | [pop-os/cosmic-comp](https://github.com/pop-os/cosmic-comp) | epoch tag |
| `cosmic-edit` | [pop-os/cosmic-edit](https://github.com/pop-os/cosmic-edit) | epoch tag |
| `cosmic-files` | [pop-os/cosmic-files](https://github.com/pop-os/cosmic-files) | epoch tag |
| `cosmic-greeter` | [pop-os/cosmic-greeter](https://github.com/pop-os/cosmic-greeter) | epoch tag |
| `cosmic-icons` | [pop-os/cosmic-icons](https://github.com/pop-os/cosmic-icons) | epoch tag |
| `cosmic-idle` | [pop-os/cosmic-idle](https://github.com/pop-os/cosmic-idle) | epoch tag |
| `cosmic-initial-setup` | [pop-os/cosmic-initial-setup](https://github.com/pop-os/cosmic-initial-setup) | epoch tag |
| `cosmic-launcher` | [pop-os/cosmic-launcher](https://github.com/pop-os/cosmic-launcher) | epoch tag |
| `cosmic-notifications` | [pop-os/cosmic-notifications](https://github.com/pop-os/cosmic-notifications) | epoch tag |
| `cosmic-osd` | [pop-os/cosmic-osd](https://github.com/pop-os/cosmic-osd) | epoch tag |
| `cosmic-panel` | [pop-os/cosmic-panel](https://github.com/pop-os/cosmic-panel) | epoch tag |
| `cosmic-player` | [pop-os/cosmic-player](https://github.com/pop-os/cosmic-player) | epoch tag |
| `cosmic-randr` | [pop-os/cosmic-randr](https://github.com/pop-os/cosmic-randr) | epoch tag |
| `cosmic-screenshot` | [pop-os/cosmic-screenshot](https://github.com/pop-os/cosmic-screenshot) | epoch tag |
| `cosmic-session` | [pop-os/cosmic-session](https://github.com/pop-os/cosmic-session) | epoch tag |
| `cosmic-settings` | [pop-os/cosmic-settings](https://github.com/pop-os/cosmic-settings) | epoch tag |
| `cosmic-settings-daemon` | [pop-os/cosmic-settings-daemon](https://github.com/pop-os/cosmic-settings-daemon) | epoch tag |
| `cosmic-store` | [pop-os/cosmic-store](https://github.com/pop-os/cosmic-store) | epoch tag |
| `cosmic-term` | [pop-os/cosmic-term](https://github.com/pop-os/cosmic-term) | epoch tag |
| `cosmic-theme-extra` | [pop-os/cosmic-theme-extra](https://github.com/pop-os/cosmic-theme-extra) | branch: master |
| `cosmic-wallpapers` | [pop-os/cosmic-wallpapers](https://github.com/pop-os/cosmic-wallpapers) | epoch tag |
| `cosmic-workspaces-epoch` | [pop-os/cosmic-workspaces-epoch](https://github.com/pop-os/cosmic-workspaces-epoch) | epoch tag |
| `pop-launcher` | [pop-os/launcher](https://github.com/pop-os/launcher) | epoch tag |
| `xdg-desktop-portal-cosmic` | [pop-os/xdg-desktop-portal-cosmic](https://github.com/pop-os/xdg-desktop-portal-cosmic) | epoch tag |

A `cosmic-desktop` meta package is also produced upon a full build, declaring dependencies on all successfully built components above.

---

## Build Dependencies

The following packages are installed automatically during the build phase unless `-skip-deps` is specified:

`build-essential`, `curl`, `git`, `libdbus-1-dev`, `libdisplay-info-dev`, `libflatpak-dev`, `libglvnd-dev`, `libgstreamer-plugins-base1.0-dev`, `libgstreamer1.0-dev`, `libinput-dev`, `libpam0g-dev`, `libpixman-1-dev`, `libseat-dev`, `libssl-dev`, `libwayland-dev`, `libxkbcommon-dev`, `lld`, `pkg-config`, `rustup`.

The `just` build tool is installed via `cargo` if not already present on the system.

---

## Project Structure

```
cosmic-deb/
├── main.go                     — Primary Go source file
├── repos.json                  — Per-repo URL and tag configuration
├── go.mod                      — Go module descriptor
├── Makefile                    — Build and installation automation
├── README.md                   — Project documentation
└── scripts/
    ├── install-local.sh        — Installs packages from a local build output
    ├── install-release.sh      — Downloads and installs packages from GitHub Releases
    └── uninstall.sh            — Removes installed COSMIC components
```

---

## Licence

GPL-2.0. Each upstream COSMIC component is subject to its own licence, most commonly the GNU General Public Licence version 3.0 (GPL-3.0). Refer to the respective upstream repositories for details.
