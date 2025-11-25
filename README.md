# **mpm** - Minecraft Plugin Manager

A CLI tool to manage Minecraft server plugins and mods using Modrinth and Hangar APIs. It allows you to install, update, and remove plugins, as well as manage the server jar itself.

## Features

- **Plugin Management**: Install, update, and remove Minecraft server plugins
- **Server Management**: Download and manage Minecraft server jars
- **Server Startup**: Start the server with custom commands and automatic configuration
- **Custom Scripts**: Define and run custom scripts (like npm scripts)
- **Startup Commands**: Automatically execute console commands when server starts
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

### Run custom scripts

```bash
# Run a script defined in package.yml
mpm run <script-name>

# Examples
mpm run backup
mpm run clean
```

### Start the server

```bash
# Start the Minecraft server with configured settings
mpm serve
```

The server will start using the command in `server.start_command` (or a default command), and will automatically execute any `startup_commands` after the server starts.

### Example package.yml

```yaml
name: my-server
version: "1.0"
server:
    type: paper
    minecraft_version: 1.20.4
    build: latest
    start_command: "java -Xms4G -Xmx8G -jar server.jar nogui"
scripts:
    backup: "tar -czf backups/backup-$(date +%Y%m%d-%H%M%S).tar.gz world world_nether world_the_end"
    clean: "rm -rf logs/*.log"
    restart: "mpm serve"
startup_commands:
    - "gamerule doMobSpawning true"
    - "difficulty normal"
    - "say Server started successfully!"
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
- Custom server start command
- Plugin versions and dependencies
- Plugin sources (Modrinth or Hangar)
- Custom scripts (like npm scripts)
- Startup commands that run when the server starts

### Custom Scripts

Similar to npm scripts, you can define custom commands in the `scripts` section of package.yml:

```yaml
scripts:
    backup: "tar -czf backups/backup-$(date +%Y%m%d-%H%M%S).tar.gz world"
    clean: "rm -rf logs/*.log"
    build: "./gradlew build"
```

Run them with: `mpm run <script-name>`

### Server Configuration

Configure how your server starts:

- **`start_command`**: Custom Java command to start the server
  - If not specified, uses default: `java -Xms2G -Xmx4G -jar server.jar nogui`
  - Customize memory, JVM flags, etc.

- **`startup_commands`**: List of console commands to run after server starts
  - Executed in order, 1 second apart
  - Useful for setting game rules, difficulty, sending messages, etc.

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