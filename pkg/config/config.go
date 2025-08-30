/*
 * Copyright (c) 2023 shenjunzheng@gmail.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"errors"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	DefaultConfigType = "json"
)

var (
	// ERROR
	ErrInvalidDirectory  = errors.New("invalid directory path")
	ErrMissingConfigName = errors.New("config name not specified")
)

type Manager struct {
	App         string
	EnvPrefix   string
	Path        string
	Name        string
	WriteConfig bool

	Viper *viper.Viper
}

// New initializes the configuration settings.
// It sets up the name, type, and path for the configuration file.
func New(app, path, name, envPrefix string, writeConfig bool) (*Manager, error) {
	if len(app) == 0 {
		return nil, ErrMissingConfigName
	}

	v := viper.New()
	v.SetConfigType(DefaultConfigType)
	var err error

	// Path
	if len(path) == 0 {
		path, err = os.UserHomeDir()
		if err != nil {
			path = os.TempDir()
		}
		path += string(os.PathSeparator) + "." + app
	}
	if err := PrepareDir(path); err != nil {
		return nil, err
	}
	v.AddConfigPath(path)

	// Name
	if len(name) == 0 {
		name = app
	}
	v.SetConfigName(name)

	// Env
	if len(envPrefix) != 0 {
		v.SetEnvPrefix(strings.ToUpper(app))
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}

	return &Manager{
		App:         app,
		EnvPrefix:   envPrefix,
		Path:        path,
		Name:        name,
		Viper:       v,
		WriteConfig: writeConfig,
	}, nil
}

// Load loads the configuration from the previously initialized file.
// It unmarshals the configuration into the provided conf interface.
func (c *Manager) Load(conf interface{}) error {
	if err := c.Viper.ReadInConfig(); err != nil {
		log.Error().Err(err).Msg("read config failed")
		if c.WriteConfig {
			if err := c.Viper.SafeWriteConfig(); err != nil {
				return err
			}
		}
	}
	if err := c.Viper.Unmarshal(conf, decoderConfig()); err != nil {
		return err
	}
	return nil
}

// LoadFile loads the configuration from a specified file.
// It unmarshals the configuration into the provided conf interface.
func (c *Manager) LoadFile(file string, conf interface{}) error {
	c.Viper.SetConfigFile(file)
	if err := c.Viper.ReadInConfig(); err != nil {
		return err
	}
	if err := c.Viper.Unmarshal(conf, decoderConfig()); err != nil {
		return err
	}
	return nil
}

// SetConfig sets a configuration key to a specified value.
// It also writes the updated configuration back to the file.
func (c *Manager) SetConfig(key string, value interface{}) error {
	c.Viper.Set(key, value)
	if c.WriteConfig {
		if err := c.Viper.WriteConfig(); err != nil {
			return err
		}
	}
	return nil
}

// GetConfig retrieves all configuration settings as a map.
func (c *Manager) GetConfig() map[string]interface{} {
	return c.Viper.AllSettings()
}

// PrepareDir ensures that the specified directory path exists.
// If the directory does not exist, it attempts to create it.
func PrepareDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !stat.IsDir() {
		log.Debug().Msgf("%s is not a directory", path)
		return ErrInvalidDirectory
	}
	return nil
}
