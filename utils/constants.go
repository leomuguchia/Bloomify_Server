// File: utils/constants.go
package utils

import "time"

// AuthCachePrefix is the prefix used for Redis authorization cache keys.
const AuthCachePrefix = "auth:"

// AuthCacheTTL is the time-to-live for authorization cache entries.
const AuthCacheTTL = 10 * time.Minute
