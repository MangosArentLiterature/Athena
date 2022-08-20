/* Athena - A server for Attorney Online 2 written in Go
Copyright (C) 2022 MangosArentLiterature <mango@transmenace.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. */

// Package settings handles reading and writing to Athena's configuration files.
package settings

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
)

// Stores the path to the config directory
var ConfigPath string

type Config struct {
	ServerConfig `toml:"Server"`
	MSConfig     `toml:"MasterServer"`
}

type ServerConfig struct {
	Addr       string `toml:"addr"`
	Port       int    `toml:"port"`
	Name       string `toml:"name"`
	Desc       string `toml:"description"`
	MaxPlayers int    `toml:"max_players"`
	MaxMsg     int    `toml:"max_message_length"`
	BufSize    int    `toml:"log_buffer_size"`
	BanLen     string `toml:"default_ban_duration"`
	LogLevel   string `toml:"log_level"`
	LogDir     string `toml:"log_directory"`
	EnableWS   bool   `toml:"enable_webao"`
	WSPort     int    `toml:"webao_port"`
	MCLimit    int    `toml:"multiclient_limit"`
	AssetURL   string `toml:"asset_url"`
	WebhookURL string `toml:"webhook_url"`
}
type MSConfig struct {
	Advertise bool   `toml:"advertise"`
	MSAddr    string `toml:"addr"`
}

// Returns a default configuration.
func defaultConfig() *Config {
	return &Config{
		ServerConfig{
			Addr:       "",
			Port:       27016,
			Name:       "Unnamed Server",
			Desc:       "",
			MaxPlayers: 100,
			MaxMsg:     256,
			BufSize:    150,
			BanLen:     "3d",
			LogLevel:   "info",
			LogDir:     "logs",
			EnableWS:   false,
			WSPort:     27017,
			MCLimit:    16,
		},
		MSConfig{
			Advertise: false,
			MSAddr:    "https://servers.aceattorneyonline.com/servers",
		},
	}
}

// Load reads the server's main configuration file.
func (conf *Config) Load() error {
	_, err := toml.DecodeFile(ConfigPath+"/config.toml", conf)
	if err != nil {
		return err
	}
	return nil
}

// GetConfig returns the server's config options.
func GetConfig() (*Config, error) {
	conf := defaultConfig()
	err := conf.Load()

	if err != nil {
		return nil, err
	}

	return conf, nil
}

// LoadMusic reads the server's music file, returning it's contents.
func LoadMusic() ([]string, error) {
	var musicList []string
	f, err := os.Open(ConfigPath + "/music.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	in := bufio.NewScanner(f)
	for in.Scan() {
		musicList = append(musicList, in.Text())
	}
	if len(musicList) == 0 {
		return nil, fmt.Errorf("empty musiclist")
	}
	if strings.ContainsRune(musicList[0], '.') {
		musicList = append([]string{"Songs"}, musicList...)
	}
	return musicList, nil
}

// LoadFile reads a server file, returning it's contents.
func LoadFile(file string) ([]string, error) {
	var l []string
	f, err := os.Open(ConfigPath + file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	in := bufio.NewScanner(f)
	for in.Scan() {
		l = append(l, in.Text())
	}
	return l, nil
}

// LoadAreas reads the server's area configuration file, returning it's contents.
func LoadAreas() ([]area.AreaData, error) {
	var conf struct {
		Area []area.AreaData
	}
	_, err := toml.DecodeFile(ConfigPath+"/areas.toml", &conf)
	if err != nil {
		return conf.Area, err
	}
	if len(conf.Area) == 0 {
		return conf.Area, fmt.Errorf("empty arealist")
	}
	return conf.Area, nil
}

// LoadAreas reads the server's role configuration file, returning it's contents.
func LoadRoles() ([]permissions.Role, error) {
	var conf struct {
		Role []permissions.Role
	}
	_, err := toml.DecodeFile(ConfigPath+"/roles.toml", &conf)
	if err != nil {
		return conf.Role, err
	}
	if len(conf.Role) == 0 {
		return conf.Role, fmt.Errorf("empty rolelist")
	}
	return conf.Role, nil
}
