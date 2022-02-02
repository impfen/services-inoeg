module github.com/kiebitz-oss/services

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.4
	github.com/jackc/pgx/v4 v4.14.1
	github.com/kiprotect/go-helpers v0.0.0-20211210144244-79ce90e73e79
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli v1.22.5
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
)

// replace github.com/kiprotect/go-helpers => ../../../../../geordi/kiprotect/go-helpers
