module github.com/GuoxinL/snowflake-gorm

go 1.16

replace github.com/bwmarrin/snowflake v0.3.0 => github.com/GuoxinL/snowflake v0.0.0-20260211023655-54c59e0cf62c

require (
	github.com/bwmarrin/snowflake v0.3.0
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/glebarez/sqlite v1.11.0
	github.com/stretchr/testify v1.8.0
	go.uber.org/atomic v1.6.0
	gorm.io/gen v0.3.26
	gorm.io/gorm v1.31.0
	gorm.io/plugin/dbresolver v1.6.2
)
