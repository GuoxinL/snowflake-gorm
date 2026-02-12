//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package route example
package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GuoxinL/snowflake-gorm"
	nodeidgorm "github.com/GuoxinL/snowflake-gorm/nodeid/gorm"
	"github.com/GuoxinL/snowflake-gorm/nodeid/gorm/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open(filepath.Join(os.TempDir(), strconv.Itoa(rand.IntN(32))+"-sqlite.db")))
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&model.SnowflakeKv{})
	if err != nil {
		panic(err)
	}

	id, err := snowflake.NewSnowflake(ctx, db, "test", 10000, time.Second, time.Second, &nodeidgorm.DefaultLogger{})
	if err != nil {
		return
	}
	for {
		fmt.Println("generate:", id.Generate())
		time.Sleep(time.Second)
	}
}
