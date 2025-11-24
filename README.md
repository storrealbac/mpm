# **mpm** - Minecraft Plugin Manager

A CLI tool to manage Minecraft server plugins and mods using the Modrinth API. It allows you to install, update, and remove plugins, as well as manage the server jar itself.

## Features

- **Plugin Management**: Install, update, and remove Minecraft server plugins
- **Server Management**: Download and manage Minecraft server jars
- **List Installed**: View all installed plugins and their versions
- **Validate**: Check for plugin updates and compatibility

## Installation

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
# Install a single plugin
mpm install <plugin-name>

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
    type: purpur
    minecraft_version: 1.20.4
    build: latest
plugins:
    - name: Fabric API
      version: 0.97.3+1.20.4
      modrinth_id: fabric-api
    - name: Sodium
      version: mc1.20.4-0.5.8
      modrinth_id: sodium
```

## Configuration

mpm uses Modrinth's API to download plugins and server jars. You can configure:

- Server type (Vanilla, Spigot, Paper, Purpur, Fabric)
- Minecraft version
- Plugin versions and dependencies

## Development

### Building

```bash
go build -o mpm main.go
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


## Support

If you encounter any issues or have questions, please open an issue on the [GitHub repository](https://github.com/storrealbac/mpm/issues).
