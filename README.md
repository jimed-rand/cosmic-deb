# cosmic-deb

`cosmic-deb` is a build orchestration tool written in Go that automates the retrieval, compilation, and packaging of the COSMIC Desktop Environment from the upstream pop-os repositories into Debian binary packages. It supports both containerised environments and direct host operation on APT-based systems.

## System Requirements

The tool supports the following distributions. For Ubuntu, only LTS and development releases are supported.

| Distribution | Supported Releases |
|--------------|-----------------------|
| Debian       | 12 (bookworm) or later |
| Ubuntu       | LTS and devel (e.g., jammy, noble, or plucky) |

## Compilation

Go 1.21 or later is required. To compile from source:

```bash
make build
```

## Configuration

Repository metadata is stored in `repos.json` (or built into the binary via `finder.go`). You can update the latest epoch tags automatically:

```bash
./cosmic-deb -update-repos
```

## Usage

### Building All Components

Root privileges are required for dependency installation and package generation.

```bash
sudo ./cosmic-deb
```

### Build Options

| Flag | Default | Description |
|------|---------|-------------|
| `-repos` | `built-in` | Path to repository configuration file |
| `-update-repos` | `false` | Fetch latest epoch tags and exit |
| `-tag` | _(empty)_ | Override tag for all repositories |
| `-workdir` | `cosmic-work` | Working directory for source and build |
| `-outdir` | `cosmic-packages` | Output directory for `.deb` packages |
| `-jobs` | CPU count | Parallel compilation jobs |
| `-skip-deps` | `false` | Skip automatic dependency installation |
| `-only` | _(empty)_ | Build a single named component |
| `-tui` | `false` | Launch interactive TUI wizard |

### Installation

To install all produced packages:

```bash
sudo dpkg -i cosmic-packages/cosmic-desktop_*.deb
sudo apt-get install -f
```

Alternatively, use the provided script:

```bash
sudo bash scripts/install-local.sh cosmic-packages
```

### Uninstallation

To remove all COSMIC components:

```bash
sudo bash scripts/uninstall.sh
```

## Technical Details

The tool retrieves source archives from GitHub and falls back to shallow `git clone` if tarball downloads fail. It detects the build system (Just, Make, or Cargo) and performs a release build with optimizations. For repositories containing a `debian/` directory, the tool uses `dpkg-buildpackage` for native packaging; otherwise, it performs a staged installation and generates the package structure manually.

Build dependencies include `build-essential`, `cmake`, `debhelper`, `devscripts`, `dh-cargo`, and various development libraries for Wayland, DBus, and graphics. Rust stable is configured via `rustup` during the build phase.

## License

`cosmic-deb` is licensed under the GPL-2.0 licence. Upstream COSMIC components are primarily licensed under the GPL-3.0 licence.
