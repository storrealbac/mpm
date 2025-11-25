# **mpm** - Minecraft Plugin Manager

A CLI tool to manage Minecraft server plugins and mods using Modrinth and Hangar APIs. It allows you to install, update, and remove plugins, as well as manage the server jar itself.

## Features

- **Plugin Management**: Install, update, and remove Minecraft server plugins
- **Server Management**: Download and manage Minecraft server jars
- **List Installed**: View all installed plugins and their versions
- **Validate**: Check for plugin updates and compatibility

## Installation

### Install

#### Windows
```powershell
powershell -c "irm https://storrealbac.github.io/mpm/install.ps1 | iex"
```

#### Linux / macOS
```bash
curl -fsSL https://storrealbac.github.io/mpm/install.sh | bash
```

That's it! mpm will be installed and ready to use.

For more information, visit [https://storrealbac.github.io/mpm](https://storrealbac.github.io/mpm)

### From Source

```bash
git clone https://github.com/storrealbac/mpm.git
cd mpm
go install
```

### Prerequisites

- Go 1.23.4 or later
- Internet connection for downloading plugins and server jars

## Usage

### Initialize a new server project

```bash
mpm init
```

This creates a `package.yml` file to manage your server configuration.

### Install plugins

```bash
# Install a single plugin (searches both Modrinth and Hangar)
mpm install <plugin-name>

# Install from a specific source
mpm install --source modrinth <plugin-name>
mpm install --source hangar <plugin-name>

# Install multiple plugins
mpm install <plugin1> <plugin2> <plugin3>

# Install from package.yml
mpm install
```

### Update plugins

```bash
# Update all plugins
mpm update

# Update specific plugin
mpm update <plugin-name>
```

### Uninstall plugins

```bash
mpm uninstall <plugin-name>
```

### List installed plugins

```bash
mpm list
```

### Validate configuration

```bash
mpm validate
```

### Example package.yml

```yaml
name: my-server
version: "1.0"
server:
    type: paper
    minecraft_version: 1.20.4
    build: latest
plugins:
    # Modrinth plugins
    - name: Vault
      version: latest
      modrinth_id: vault
    # Hangar plugins (use owner/slug format)
    - name: Geyser
      version: latest
      hangar_id: GeyserMC/Geyser-Spigot
    - name: ViaVersion
      version: latest
      hangar_id: ViaVersion/ViaVersion
```

## Configuration

mpm uses Modrinth and Hangar APIs to download plugins and server jars. You can configure:

- Server type (Paper, Purpur, Folia, Spigot, Bukkit, Sponge, Velocity, Waterfall)
- Minecraft version
- Plugin versions and dependencies
- Plugin sources (Modrinth or Hangar)

### Plugin Sources

mpm supports two plugin repositories:

- **Modrinth**: General-purpose mod and plugin repository
  - Use `modrinth_id` field in package.yml
  - Example: `modrinth_id: vault`

- **Hangar**: PaperMC's official plugin repository
  - Use `hangar_id` field in package.yml with format `owner/slug`
  - Example: `hangar_id: GeyserMC/Geyser-Spigot`
  - Specifically optimized for Paper ecosystem plugins

When using `mpm install <plugin-name>`, the tool will automatically search both repositories unless you specify `--source` flag.

## Development

### Building

```bash
go build -o mpm main.go
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


## Support

If you encounter any issues or have questions, please open an issue on the [GitHub repository](https://github.com/storrealbac/mpm/issues).