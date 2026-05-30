package config

import (
	"os"
	"strings"
)

type AppEnv string

const (
	AppEnvDevelopment AppEnv = "development"
	AppEnvTest        AppEnv = "test"
	AppEnvStaging     AppEnv = "staging"
	AppEnvPreview     AppEnv = "preview"
	AppEnvProduction  AppEnv = "production"
)

func CurrentAppEnv() AppEnv {
	return NormalizeAppEnv(os.Getenv("APP_ENV"))
}

func NormalizeAppEnv(value string) AppEnv {
	return AppEnv(strings.ToLower(strings.TrimSpace(value)))
}

func (e AppEnv) String() string {
	return string(e)
}

func (e AppEnv) IsProduction() bool {
	return e == AppEnvProduction
}

func (e AppEnv) IsDevelopment() bool {
	return e == AppEnvDevelopment
}

func (e AppEnv) IsStrictSecurity() bool {
	switch e {
	case AppEnvProduction, AppEnvStaging, AppEnvPreview:
		return true
	default:
		return false
	}
}

func (e AppEnv) AllowsDevBypass() bool {
	switch e {
	case AppEnvDevelopment, AppEnvTest:
		return true
	default:
		return false
	}
}
