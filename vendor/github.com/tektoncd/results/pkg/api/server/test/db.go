// Copyright 2020 The Tekton Authors
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

package test

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Inject sqlite error checking.
	_ "github.com/tektoncd/results/pkg/api/server/db/errors/sqlite"
)

// NewDB set up a temporary database for testing
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Create a temporary file
	tmpfile, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatalf("failed to create temp file for db: %v", err)
	}
	t.Log("test database: ", tmpfile.Name())
	t.Cleanup(func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	})

	// Connect to sqlite DB manually to load in schema.
	db, err := sql.Open("sqlite3", tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}
	defer db.Close()

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	schema, err := ioutil.ReadFile(path.Join(basepath, "../../../../schema/results.sql"))
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}
	// Create result table using the checked in scheme to ensure compatibility.
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("failed to execute the result table creation statement statement: %v", err)
	}

	// Reopen DB using gorm to use all the nice gorm tools.
	gdb, err := gorm.Open(sqlite.Open(tmpfile.Name()), &gorm.Config{
		// Configure verbose db logging to use testing logger.
		// This will show all SQL statements made if the test fails.
		Logger: logger.New(&testLogger{t: t}, logger.Config{
			LogLevel: logger.Info,
			Colorful: true,
		}),
	})
	if err != nil {
		t.Fatalf("failed to open the results.db: %v", err)
	}

	// Enable foreign key support. Only needed for sqlite instance we use for
	// tests.
	gdb.Exec("PRAGMA foreign_keys = ON;")

	return gdb
}

type testLogger struct {
	t *testing.T
}

func (t *testLogger) Printf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
