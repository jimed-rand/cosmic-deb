# cosmic-deb

**A comprehensive build framework by James Ed Randson (jimed-rand/jimedrand) for creating the Debian/Ubuntu `.deb` packages for the COSMIC Desktop Environment, built upon the foundational packaging schemas of hepp3n.**

This repository presents a comprehensive build automation tool, meticulously designed for the compilation and packaging of the COSMIC Desktop Environment into native Debian and Ubuntu `.deb` package formats. The underlying architecture and packaging sources are derived from and maintained by [hepp3n on Codeberg](https://codeberg.org/hepp3n), serving as the foundational framework for this build suite. The primary objective of this utility is to standardise the deployment process, ensuring a reproducible and structured method for integrating the COSMIC ecosystem into diverse Debian-based operating systems.

The software addresses the inherent complexities of cross-distribution package management by automating the retrieval, compilation, and assembly phases of the software development life cycle. By facilitating an interactive build mechanism, it significantly reduces the cognitive load on developers and system administrators. The tool dynamically resolves build dependencies, adapts to systemic containerised environments, and guarantees that the resulting compiled binaries adhere to rigorous structural paradigms. The methodology employed herein guarantees stability, mitigates dependency conflicts, and optimises the dissemination of the COSMIC Desktop Environment.

Furthermore, this utility incorporates advanced logic to ascertain the current state of the host system, employing native queries such as `dpkg-query` to rigorously analyse package statuses. This ensures that redundant operations are minimised and that the build environment remains pristine. The implementation leverages concurrent processing to maximise computational efficiency during the build phase, demonstrating an optimal utilisation of available hardware resources. Through this systematic approach, the build framework provides an academic and robust solution for the seamless distribution of contemporary desktop computing environments within the Linux ecosystem.

## Prerequisites

To ensure the successful execution of the build process, the host system must satisfy the following prerequisites:

- **Operating System:** Debian 12 (Bookworm) or a subsequent release, or alternatively, Ubuntu 22.04 (Jammy Jellyfish) or a subsequent LTS release. Also applied with the Developmental branch of Ubuntu (`devel`). **Note:** Non-LTS iterations of Ubuntu are explicitly unsupported due to their ephemeral lifecycle and inherent unsuitability for stable deployment environments.
- **Core Utilities:** The presence of `apt-get`, `git`, `curl`, `fakeroot`, and `dpkg-dev` is mandatory for dependency resolution and package assembly.
- **Compiler:** Go version 1.24 or later is requisite for the initial compilation of the builder itself.

## Compilation of the Builder

The initialisation phase requires the compilation of the Go-based build framework. Execute the following sequence within the terminal:

```sh
make build
```

## Operational Methodology

### Interactive Command Line Interface (CLI)

To invoke the standard interactive interface, execute the compiled binary:

```sh
./cosmic-deb
```

Upon execution, the operator is prompted to designate a source acquisition strategy, specifically selecting between a versioned epoch tag or the current *HEAD* of the primary branch. The system subsequently conducts a comprehensive validation of build dependencies; package installations are restricted strictly to absent dependencies, prompting for elevated privileges via `sudo` exclusively when operating as a non-privileged user. The resulting `.deb` packages automatically attribute the maintainer field to **hepp3n**, acknowledging their contribution as the upstream packaging author.

To systematically categorise and prevent nomenclature collisions across distinct operating system iterations, the generated packages are appended with the distribution's codename (for example, `cosmic-comp_1.0.0~noble_amd64.deb`). After the comprehensive assembly of all constituent components, the system autonomously performs a systematic cleanup, purging the source and staging directories. Subsequently, if the operational context is identified as a bare-metal or conventional virtual machine environment rather than an isolated container (such as Docker or LXC), the operator is offered a direct installation pathway for the compiled packages.

### Verbose Logging

For developers and power users who require detailed insight into every discrete step of the build pipeline, the `-verbose` flag can be appended to any invocation:

```sh
./cosmic-deb -verbose
```

When verbose mode is active, the builder emits timestamped `[VERBOSE]` log lines for every internal decision point, including: resolved flag values at startup, detected CPU core count, individual dependency check results, archive download URLs, extraction paths, build system detection per component, staging directory validation, version resolution strategy, and cleanup operations. This level of transparency is particularly useful for diagnosing build failures, understanding the exact sequence of actions taken, and auditing the build environment.

### Terminal User Interface (TUI)

For a graphically augmented interactive experience within the terminal, append the requisite flag:

```sh
./cosmic-deb -tui
```

This instantiation provides a structured, full-screen navigational wizard. The interface systematically guides the operator through the configuration of source modes, designation of output directories, adjustment of concurrency levels, and the management of dependencies. Furthermore, it incorporates a real-time monitoring module to observe the compilation trajectory.

### Command Line Variables

| Variable | Default Value | Description |
|---|---|---|
| `-tui` | `false` | Initialises the Terminal User Interface (TUI) wizard in lieu of standard CLI prompts. |
| `-repos` | `built-in` | Specifies the file path to the repository JSON configuration or utilises the `built-in` schema. |
| `-tag` | *(null)* | Mandates a global override of the epoch tag across all targeted repositories. |
| `-use-branch` | `false` | Instructs the builder to fetch and compile from the primary branch *HEAD*. |
| `-workdir` | `cosmic-work` | Designates the designated directory for transient source code and staging files. |
| `-outdir` | `cosmic-packages` | Specifies the output directory for the finalised `.deb` package archives. |
| `-jobs` | *(nproc)* | Defines the parameter for concurrent compilation tasks, optimising CPU utilisation. |
| `-skip-deps` | `false` | Bypasses the initial dependency verification and installation phase. |
| `-only` | *(null)* | Isolates the compilation process to a singular, explicitly named component. |
| `-update-repos` | `false` | Contacts upstream remote repositories to fetch recent epoch tags and overwrites the configuration. |
| `-gen-config` | `false` | Extracts the internal configuration and exports it to a `repos.json` file. |
| `-dev-finder` | `false` | Facilitates developer operations by regenerating `pkg/repos/finder.go` from the active schema. |
| `-verbose` | `false` | Enables verbose timestamped logging for all internal build decisions and operations. |

### Makefile Directives

```sh
make run                    # Executes the primary pipeline with interactive source designation
make run-verbose            # Executes the primary pipeline with verbose logging enabled
make run-tui                # Executes the primary pipeline accompanied by the TUI wizard
make run-branch             # Initiates compilation exclusively from the primary branch HEAD
make run-only COMPONENT=cosmic-term
make run-skip-deps          # Bypasses dependency validation (presumes requisite packages exist)
make update-repos           # Synchronises with upstream to register the latest epoch tags
make install                # Strategically deploys the binary executable and associated scripts to /usr/local
make uninstall              # Eradicates the installed assets from the system hierarchy
make clean                  # Purges the designated working directories and compiled binary
```

## Source Acquisition Strategies

**Epoch Tags** represent stable, formally versioned iterations of the COSMIC Desktop Environment releases (e.g., `epoch-1.0.0`). Distinct repositories may maintain independent tagging schemas, or a unified tag may be forcibly applied through the `-tag` directive.

**Branch HEAD** indicates a dynamic acquisition strategy, targeting the latest unversioned commit from the primary branch of each repository. This methodology is inherently experimental and susceptible to instability.

## Build Procedure Framework

1. **Dependency Validation:** The builder evaluates the system for missing APT packages and undertakes installation (invoking `sudo` conditionally). The integration of the Rust toolchain and the `just` command runner is strictly enforced.
2. **Sequential Ordering:** Prior to compilation, components are subjected to a structural A–Z sortation, thereby mitigating potential discrepancies arising from unpredictable build sequences.
3. **Component Processing:** For each designated component, the source material is acquired (prioritising tarball extraction with a fallback to `git clone`). If a `justfile` vendor target is detected, dependencies are vendored, followed by systematic compilation and output validation prior to the staging phase.
4. **Package Assembly:** A standardised `DEBIAN/control` manifest is generated, enumerating necessary runtime dependencies. Subsequently, the `fakeroot dpkg-deb` utility executes the synthesis of the `.deb` archive. Appended filenames rigorously reflect the host distribution's codename.
5. **Meta-package Synthesis:** The `cosmic-desktop` meta-package is algorithmically constructed to serve as an aggregate dependency linking all independently built components, simplifying holistic installation.
6. **Data Sanitisation:** Upon the successful amalgamation of all packages, the system instigates an exhaustive cleanup procedure, removing transient source hierarchies and staging constructs.
7. **Deployment Resolution:** Provided the process operates outside a constrained containerised environment, the builder consults the operator regarding the immediate system-wide deployment of the synthesised packages.

## Deployment Scripts

| Script File | Functional Purpose |
|---|---|
| `scripts/install-local.sh [dir]` | Facilitates the systemic installation of locally synthesised analytical packages from a specified directory. |
| `scripts/install-release.sh [tag]` | Orchestrates the retrieval and deployment of a verified release directly from GitHub. |
| `scripts/uninstall.sh` | Executes a comprehensive removal of the COSMIC package ecosystem from the host system. |

## Architectural Repository Structure

```
cosmic-deb/
├── main.go                    # Entry point: flag parsing, orchestration logic, verbose logging
├── go.mod / go.sum            # Go module dependency manifest configurations
├── Makefile                   # Methodological build and execution directives
├── README.md                  # Comprehensive academic documentation
├── pkg/
│   ├── build/
│   │   ├── compile.go         # Algorithmic compilation, vendoring, and staging installation
│   │   ├── deps.go            # Evaluative dependency resolution and Rust toolchain provisioning
│   │   ├── source.go          # Data acquisition mechanics via tarball or git version control
│   │   └── version.go         # Implementation of systemic version detection heuristics
│   ├── debian/
│   │   └── package.go         # Mechanisms for .deb synthesis and meta-package construction
│   ├── distro/
│   │   ├── deps.go            # Distribution-specific dependency mapping logic
│   │   └── detect.go          # Methodologies for distribution identification and container heuristics
│   ├── repos/
│   │   ├── finder.go          # Native repository enumeration (hepp3n/Codeberg)
│   │   ├── loader.go          # Configuration ingestion, epoch tag querying, and state mutation
│   │   └── types.go           # Structural definitions for repositories and related configurations
│   └── tui/
│       ├── monitor.go         # Real-time concurrent build progress observatory
│       └── wizard.go          # Algorithmic configuration solicitation interface
└── scripts/
    ├── install-local.sh
    ├── install-release.sh
    └── uninstall.sh
```

## Legal Disclaimer

This utility is provided strictly for academic and operational automation purposes. The project is an independent build framework conceptualised and coded by James Ed Randson, deriving its fundamental architectural layout from hepp3n. It is not officially affiliated with, endorsed by, or intrinsically linked to System76, Inc., the principal architects of the COSMIC Desktop Environment. The build mechanism employed herein is fundamentally designed to facilitate existing upstream packaging schemas without infringing upon established intellectual property rights or software distribution guidelines. It does not intend to subvert, misappropriate, or violate any established terms of service or proprietary constraints. All assets retrieved and compiled by this tool remain the intellectual property of their respective creators.

## Licensing Framework

### The Builder Project

The `cosmic-deb` build automation tool and its associated source code within this repository are distributed under open-source provisions. Licensed under the GNU General Public License v2.0 or later (GPL-2.0-or-later). Operators and contributors are granted the freedom to study, modify, and redistribute this utility in accordance with standard open-source stipulations.

### COSMIC Desktop Environment

The COSMIC Desktop Environment, its constituent components, and the original source code are the intellectual property of **System76, Inc.** and are subject to the licensing models explicitly designated within their respective upstream repositories (predominantly the GNU General Public Licence v3.0, Mozilla Public Licence 2.0, or equivalent open-source licences). Packaging schemas leveraged by this tool inherit the intellectual provisions defined by their authors. All operators must rigidly comply with the licensing terms decreed by System76 and upstream contributors when utilising or distributing compiled binaries.

## Expanded Acknowledgements

The realisation of this project owes a profound debt of gratitude to the wider open-source community. Specific, formal recognition is extended to the following entities:

1. **James Ed Randson**: As the primary developer of this builder utility, responsible for the automated Go-based workflow, abstraction logic, and concurrent execution pipelines.
2. **hepp3n**: The seminal upstream architectural framework for Debian packaging, which underpins the fundamental logic of this build tool, was meticulously crafted and is maintained by **hepp3n**. Their foundational work available at [codeberg.org/hepp3n](https://codeberg.org/hepp3n) is rigorously integral to the successful deployment of the COSMIC ecosystem on Debian-based derivatives. We formally acknowledge their paramount effort as the primary packaging author.
3. **System76, Inc.**: For their pioneering work in engineering the COSMIC Desktop Environment, advancing contemporary Rust-based desktop paradigms, and dedicating extensive resources to the broader open-source Linux community.
4. **The Free and Open Source Software (FOSS) Ecosystem**: Extends to the maintainers of the Go programming language, the Debian Project, and Canonical Ltd., whose underlying systems, comprehensive libraries, and compilation utilities constitute the fundamental architecture upon which this logic fundamentally depends.
