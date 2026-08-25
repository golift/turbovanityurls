// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:forbidigo
package service

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golift.io/badgedata"
	_ "golift.io/badgedata/grafana" // we use grafana here.
	"golift.io/turbovanityurls/pkg/handler"
	yaml "gopkg.in/yaml.v3"
)

// Flags are the CLI flags.
type Flags struct {
	ListenAddr string
	Timeout    time.Duration
	ConfigPath string
	ShowVer    bool
}

// Config is the application config: vanity paths plus optional badgedata.
type Config struct {
	*handler.Config `yaml:",inline"`

	BDPath string `yaml:"bd_path,omitempty"`
	flags  *Flags
}

const defaultTimeout = 15 * time.Second

// ParseFlags parses CLI flags from args.
func ParseFlags(args []string) *Flags {
	flag := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f := &Flags{ListenAddr: ":" + os.Getenv("PORT")}

	if f.ListenAddr == ":" {
		f.ListenAddr = ":8080"
	}

	flag.DurationVar(&f.Timeout, "t", defaultTimeout, "HTTP request timeout")
	flag.StringVar(&f.ListenAddr, "l", f.ListenAddr, "HTTP server listen address")
	flag.StringVar(&f.ConfigPath, "c", DefaultConfFile, "config file path")
	flag.BoolVar(&f.ShowVer, "v", false, "show version and exit")

	flag.Usage = func() {
		fmt.Println("Usage: turbovanityurls [-c <config-file>] [-l <listen-address>] [-t <timeout>]")
		flag.PrintDefaults()
	}

	_ = flag.Parse(args)

	return f
}

// Setup loads the config file and registers HTTP handlers.
func Setup(flags *Flags) (*Config, error) {
	config := &Config{flags: flags}

	err := config.ParseConfig(flags.ConfigPath)
	if err != nil {
		return nil, err
	}

	vanityHandler, err := handler.New(config.Config)
	if err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	if config.BDPath != "" {
		http.Handle(config.BDPath, badgedata.Handler())
	}

	http.Handle("/", vanityHandler)

	return config, nil
}

// ParseConfig reads and unmarshals the YAML config file.
func (c *Config) ParseConfig(configPath string) error {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) && configPath == DefaultConfFile {
		log.Printf("Default Config File Not Found: %s - trying ./config.yaml", configPath)
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // expected file inclusion.
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("unmarshaling config file: %w", err)
	}

	if c.Title == "" {
		c.Title = c.Host
	}

	if c.BDPath != "" && !strings.HasSuffix(c.BDPath, "/") {
		c.BDPath += "/"
	}

	return nil
}

// Start runs the HTTP server until it exits.
func (c *Config) Start() error {
	if strings.HasPrefix(c.flags.ListenAddr, ":") {
		// A message so you know when it's started; a clickable link for dev'ing.
		log.Println("Listening at http://127.0.0.1" + c.flags.ListenAddr)
	}

	server := &http.Server{
		Addr:              c.flags.ListenAddr,
		ReadHeaderTimeout: c.flags.Timeout,
	}

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("web server problem: %w", err)
	}

	return nil
}
