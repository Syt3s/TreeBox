// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/syt3s/TreeBox/internal/branding"
)

// File is the configuration object.
var File *ini.File

func Init() error {
	configFile := os.Getenv(branding.ConfigPathEnvVar)
	if configFile == "" {
		configFile = os.Getenv(branding.LegacyConfigPathEnvVar)
	}
	if configFile == "" {
		configFile = "configs/app.ini"
		if _, err := os.Stat(configFile); err != nil {
			if os.IsNotExist(err) {
				configFile = "conf/app.ini"
			} else {
				return errors.Wrap(err, "stat config file")
			}
		}
	}

	var err error
	File, err = ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
	}, configFile)
	if err != nil {
		return errors.Wrapf(err, "parse %q", configFile)
	}

	if err := File.Section("app").MapTo(&App); err != nil {
		return errors.Wrap(err, "map 'server'")
	}

	if App.ExternalURL == "" {
		return errors.New("app.external_url is required")
	}
	App.ExternalURL = strings.TrimRight(App.ExternalURL, "/")

	if err := File.Section("server").MapTo(&Server); err != nil {
		return errors.Wrap(err, "map 'server'")
	}

	if err := File.Section("database").MapTo(&Database); err != nil {
		return errors.Wrap(err, "map 'database'")
	}

	if err := File.Section("redis").MapTo(&Redis); err != nil {
		return errors.Wrap(err, "map 'redis'")
	}

	if err := File.Section("recaptcha").MapTo(&Recaptcha); err != nil {
		return errors.Wrap(err, "map 'recaptcha'")
	}

	if err := File.Section("mail").MapTo(&Mail); err != nil {
		return errors.Wrap(err, "map 'mail'")
	}

	if err := File.Section("pixel").MapTo(&Pixel); err != nil {
		return errors.Wrap(err, "map 'pixel'")
	}

	if err := File.Section("upload").MapTo(&Upload); err != nil {
		return errors.Wrap(err, "map 'upload'")
	}

	serviceSections := File.Section("service").ChildSections()
	for _, serviceSection := range serviceSections {
		serviceSection := serviceSection
		var backend struct {
			Prefix     string `ini:"prefix"`
			ForwardURL string `ini:"forward_url"`
		}
		if err := serviceSection.MapTo(&backend); err != nil {
			return errors.Wrapf(err, "map 'service.%s'", serviceSection.Name())
		}
		Service.Backends = append(Service.Backends, backend)
	}

	return nil
}
