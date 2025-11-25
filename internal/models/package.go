package models

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Package struct {
	Name    string       `yaml:"name"`
	Version string       `yaml:"version"`
	Server  ServerConfig `yaml:"server,omitempty"`
	Plugins []Plugin     `yaml:"plugins"`
}

type ServerConfig struct {
	Type             string `yaml:"type"`              // paper, purpur, folia, spigot, bukkit, sponge, velocity, waterfall
	MinecraftVersion string `yaml:"minecraft_version"` // 1.20.1, etc.
	Build            string `yaml:"build,omitempty"`   // latest or specific build number
}

type Plugin struct {
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`     // Version específica requerida
	ModrinthID   string   `yaml:"modrinth_id"` // ID o Slug de Modrinth
	Optional     bool     `yaml:"optional,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

// PackageLock stores checksums and resolved versions
type PackageLock struct {
	Plugins map[string]PluginLock `yaml:"plugins"` // Key is ModrinthID
}

type PluginLock struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Hash    string `yaml:"hash"` // SHA512 hash
}

// LoadPackageFromFile carga un package.yml desde archivo
func LoadPackageFromFile(filename string) (*Package, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var pkg Package
	err = yaml.Unmarshal(data, &pkg)
	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

// SaveToFile guarda el package a un archivo YAML
func (p *Package) SaveToFile(filename string) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadPackageLockFromFile loads package-lock.yml
func LoadPackageLockFromFile(filename string) (*PackageLock, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		// If file doesn't exist, return empty lock
		if os.IsNotExist(err) {
			return &PackageLock{
				Plugins: make(map[string]PluginLock),
			}, nil
		}
		return nil, err
	}

	var lock PackageLock
	err = yaml.Unmarshal(data, &lock)
	if err != nil {
		return nil, err
	}

	if lock.Plugins == nil {
		lock.Plugins = make(map[string]PluginLock)
	}

	return &lock, nil
}

// SaveToFile saves the lock file
func (l *PackageLock) SaveToFile(filename string) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
