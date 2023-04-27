package internal

import "time"

const (
	PasswordExpiredDuration = 30 * 24 * time.Hour
	EmailCodeExpireTime     = 5 * time.Minute
	API_VERSION             = "/1.0"
	API_PREFIX              = "/api"
	ADMINAPI_PREFIX         = "/admin"

	SYSTEM_API_VERSION = "/1.0"
	SYSTEM_API_PREFIX  = "/system-api"
)