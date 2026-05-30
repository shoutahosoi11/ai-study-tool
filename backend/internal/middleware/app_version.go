package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	appVersionHeader  = "X-App-Version"
	appPlatformHeader = "X-Platform"

	ContextAppVersionKey  = "app_version"
	ContextAppPlatformKey = "app_platform"

	AppPlatformIOS     = "ios"
	AppPlatformAndroid = "android"
)

type AppVersionMiddleware struct {
	minSupportedVersions       map[string]string
	rejectMissingMobileHeaders bool
}

type AppVersionConfig struct {
	AppEnv                     string
	MinSupportedIOSVersion     string
	MinSupportedAndroidVersion string
	RejectMissingMobileHeaders bool
}

func NewAppVersionMiddlewareFromEnv(appEnv string) *AppVersionMiddleware {
	appEnv = strings.TrimSpace(appEnv)
	return NewAppVersionMiddleware(AppVersionConfig{
		AppEnv:                     appEnv,
		MinSupportedIOSVersion:     os.Getenv("MIN_SUPPORTED_IOS_VERSION"),
		MinSupportedAndroidVersion: os.Getenv("MIN_SUPPORTED_ANDROID_VERSION"),
		RejectMissingMobileHeaders: appconfig.NormalizeAppEnv(appEnv).IsProduction(),
	})
}

func NewAppVersionMiddleware(config AppVersionConfig) *AppVersionMiddleware {
	return &AppVersionMiddleware{
		minSupportedVersions: map[string]string{
			AppPlatformIOS:     strings.TrimSpace(config.MinSupportedIOSVersion),
			AppPlatformAndroid: strings.TrimSpace(config.MinSupportedAndroidVersion),
		},
		rejectMissingMobileHeaders: config.RejectMissingMobileHeaders,
	}
}

func (m *AppVersionMiddleware) Check(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeMobile {
			return next(c)
		}

		// Version headers are client-provided metadata. They are useful for
		// rollout gating only after Mobile authentication and App Check have
		// already bound the request to a legitimate app instance.
		platform := strings.ToLower(strings.TrimSpace(c.Request().Header.Get(appPlatformHeader)))
		version := strings.TrimSpace(c.Request().Header.Get(appVersionHeader))
		if platform != "" && platform != AppPlatformIOS && platform != AppPlatformAndroid {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid platform")
		}
		if platform == "" || version == "" {
			if m.rejectMissingMobileHeaders {
				return echo.NewHTTPError(http.StatusBadRequest, "missing mobile app metadata")
			}
			return next(c)
		}
		if _, err := parseVersion(version); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid app version")
		}

		c.Set(ContextAppPlatformKey, platform)
		c.Set(ContextAppVersionKey, version)

		minVersion := strings.TrimSpace(m.minSupportedVersions[platform])
		if minVersion == "" {
			return next(c)
		}
		if _, err := parseVersion(minVersion); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "mobile version policy unavailable")
		}
		if compareVersions(version, minVersion) < 0 {
			return c.JSON(http.StatusUpgradeRequired, map[string]string{
				"error":      "upgrade_required",
				"minVersion": minVersion,
				"platform":   platform,
			})
		}

		return next(c)
	}
}

func GetAppVersion(c echo.Context) (string, bool) {
	version, ok := c.Get(ContextAppVersionKey).(string)
	return version, ok && version != ""
}

func GetAppPlatform(c echo.Context) (string, bool) {
	platform, ok := c.Get(ContextAppPlatformKey).(string)
	return platform, ok && platform != ""
}

func compareVersions(current string, minimum string) int {
	currentParts, currentErr := parseVersion(current)
	minimumParts, minimumErr := parseVersion(minimum)
	if currentErr != nil || minimumErr != nil {
		return -1
	}
	for i := 0; i < len(currentParts); i++ {
		if currentParts[i] > minimumParts[i] {
			return 1
		}
		if currentParts[i] < minimumParts[i] {
			return -1
		}
	}
	return 0
}

func parseVersion(value string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return result, strconv.ErrSyntax
	}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return result, strconv.ErrSyntax
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return result, strconv.ErrSyntax
		}
		result[i] = parsed
	}
	return result, nil
}
