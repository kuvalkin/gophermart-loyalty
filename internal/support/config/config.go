package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	RunAddress            string `env:"RUN_ADDRESS"`
	DatabaseDSN           string `env:"DATABASE_URI"`
	DatabaseTimeout       time.Duration
	AccrualSystemAddress  string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	AccrualMaxRetries     int
	AccrualMaxRetryPeriod time.Duration
	AccrualTimeout        time.Duration
	MinPasswordLength     int
	TokenExpirationPeriod time.Duration
}

func Resolve() (*Config, error) {
	conf := &Config{
		RunAddress:           "localhost:8080",
		DatabaseDSN:          "",
		AccrualSystemAddress: "",
		// hardcoded for now
		AccrualMaxRetries:     10,
		AccrualMaxRetryPeriod: 5 * time.Minute,
		AccrualTimeout:        time.Minute,
		DatabaseTimeout:       5 * time.Second,
		MinPasswordLength:     12,
		TokenExpirationPeriod: time.Hour,
	}

	parseFlags(conf)

	err := parseEnv(conf)
	if err != nil {
		return nil, fmt.Errorf("error parsing env: %w", err)
	}

	err = checkRequiredFields(conf)
	if err != nil {
		return nil, fmt.Errorf("required fields: %w", err)
	}

	return conf, nil
}

func parseFlags(conf *Config) {
	flag.Func(
		"a",
		fmt.Sprintf("Address where server will be started, host:port (default %v)", conf.RunAddress),
		func(addr string) error {
			err := validateServerAddress(addr)
			if err != nil {
				return fmt.Errorf("invalid server address: %w", err)
			}

			conf.RunAddress = addr

			return nil
		},
	)
	flag.Func(
		"r",
		"Base url for accrual system, http(s)://host:port",
		func(u string) error {
			err := validateAccrualSystemAddress(u)
			if err != nil {
				return fmt.Errorf("invalid accrual system address: %w", err)
			}

			conf.AccrualSystemAddress = u

			return nil
		},
	)
	flag.StringVar(&conf.DatabaseDSN, "d", conf.DatabaseDSN, "Database DSN for PostgreSQL connection")

	flag.Parse()
}

func parseEnv(conf *Config) error {
	if err := env.Parse(conf); err != nil {
		return fmt.Errorf("parse environment variables: %w", err)
	}

	err := validateServerAddress(conf.RunAddress)
	if err != nil {
		return fmt.Errorf("invalid server address: %w", err)
	}

	if conf.AccrualSystemAddress != "" {
		err = validateAccrualSystemAddress(conf.AccrualSystemAddress)
		if err != nil {
			return fmt.Errorf("invalid accrual system address: %w", err)
		}
	}

	return nil
}

func checkRequiredFields(conf *Config) error {
	if conf.RunAddress == "" {
		return errors.New("run address is required")
	}

	if conf.DatabaseDSN == "" {
		return errors.New("database DSN is required")
	}

	if conf.AccrualSystemAddress == "" {
		return errors.New("accrual system address is required")
	}

	return nil
}

func validateServerAddress(address string) error {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return errors.New("need address in a form host:port")
	}

	if err := validateIP(parts[0]); err != nil {
		return fmt.Errorf("invalid ip address: %w", err)
	}

	if err := validatePort(parts[1]); err != nil {
		return fmt.Errorf("invalid port in address: %w", err)
	}

	return nil
}

func validateIP(ipString string) error {
	if ipString == "localhost" {
		return nil
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("could not parse ip: %v", ipString)
	}

	return nil
}

func validatePort(portString string) error {
	port, err := strconv.Atoi(portString)
	if err != nil {
		return fmt.Errorf("could not parse port: %w", err)
	}

	if port < 0 || port > 65535 {
		return fmt.Errorf("port out of range: %d", port)
	}

	return nil
}

func validateAccrualSystemAddress(urlString string) error {
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("could not parse url: %w", err)
	}

	if u.Scheme == "" {
		return fmt.Errorf("url scheme is required")
	}

	if u.Host == "" {
		return fmt.Errorf("url host is required")
	}

	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("url should not contain path, query or fragment")
	}

	return nil
}
