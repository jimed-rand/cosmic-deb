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

## Usage

### Building All COSMIC Components

The following command initiates the full build pipeline. Root privileges are required to invoke `apt-get` for dependency installation.

```bash
sudo ./cosmic-deb
```

When launched without the `-tag` flag, the tool uses `git ls-remote` to enumerate available `epoch-*` tags directly from the upstream `cosmic-epoch` repository without requiring a GitHub API token. The available tags are presented in an interactive selection prompt. To bypass the prompt entirely, specify the desired tag directly:

```bash
sudo ./cosmic-deb -tag epoch-1.0.7
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
| `-tag` | _(empty)_ | The upstream COSMIC epoch release tag (e.g. `epoch-1.0.7`). When omitted, available tags are fetched and presented interactively. |
| `-workdir` | `cosmic-work` | Working directory for source checkout and compilation |
| `-outdir` | `cosmic-packages` | Output directory for the produced `.deb` packages |
| `-jobs` | CPU count | Number of parallel compilation jobs |
| `-skip-deps` | `false` | Skip automatic installation of build dependencies |
| `-only` | _(empty)_ | Restrict the build to a single named `cosmic-*` component |
| `-tui` | `false` | Launch the interactive TUI wizard and build monitor |

### Usage Examples

```bash
sudo ./cosmic-deb -tag epoch-1.0.7 -jobs 8 -outdir /tmp/debs

sudo ./cosmic-deb -tag epoch-1.0.7 -only cosmic-term -skip-deps

sudo ./cosmic-deb -workdir /mnt/build -outdir /mnt/debs
```

---

## How Source Archives Are Fetched

Available epoch tags are retrieved by running `git ls-remote --tags` against the upstream `cosmic-epoch` repository. This approach requires no GitHub API token and is not subject to API rate limiting.

For each COSMIC component, the source archive is downloaded directly from GitHub as a `.tar.gz` tarball using the selected epoch release tag:

```
https://github.com/pop-os/<component>/archive/refs/tags/<tag>.tar.gz
```

The tool uses `tar -tzf` to inspect the archive and determine the exact top-level directory name before extraction, eliminating guesswork around GitHub's naming convention. The archive is removed after successful extraction.

---

## Installation

### From a Local Build

Upon successful completion of the build pipeline, produced `.deb` packages reside in the designated output directory. To install the entire COSMIC desktop environment using the generated meta package:

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

The release installation script retrieves packages matching the host system's architecture (`amd64` or `arm64`) from the GitHub Releases API and installs them using `dpkg`.

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
| `run-only` | Build a single component (requires `COMPONENT=<name>`) |
| `fmt` | Format Go source files using `go fmt` |
| `vet` | Run static analysis using `go vet` |
| `tidy` | Tidy the Go module dependency graph |
| `help` | Display usage information for all targets |

### Makefile Variables

The following variables may be overridden on the command line:

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

**Example — installing from a release within a container:**

```bash
docker run --rm -it \
  debian:bookworm bash -c \
  "apt-get update && apt-get install -y curl && \
   curl -fsSL https://raw.githubusercontent.com/jimed-rand/cosmic-deb/main/scripts/install-release.sh | bash"
```

---

## Components Built

The following upstream repositories from the [pop-os](https://github.com/pop-os) organisation are included in the build pipeline:

- `cosmic-applets`
- `cosmic-applibrary`
- `cosmic-bg`
- `cosmic-comp`
- `cosmic-edit`
- `cosmic-files`
- `cosmic-greeter`
- `cosmic-icons`
- `cosmic-idle`
- `cosmic-launcher`
- `cosmic-notifications`
- `cosmic-osd`
- `cosmic-panel`
- `cosmic-player`
- `cosmic-randr`
- `cosmic-screenshot`
- `cosmic-session`
- `cosmic-settings`
- `cosmic-settings-daemon`
- `cosmic-store`
- `cosmic-term`
- `cosmic-theme-extra`
- `cosmic-wallpapers`
- `cosmic-workspaces-epoch`
- `xdg-desktop-portal-cosmic`

A `cosmic-desktop` meta package is also produced upon a full build, declaring dependencies on all of the above components.

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
