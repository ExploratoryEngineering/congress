package dbstore

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/congress/utils"
	"github.com/ExploratoryEngineering/logging"
)

// PostgresEmbedded embeds PostgreSQL and allows it to be controlled from
// f.e. an unit test. Note that the /tmp directory in OS X fills up rather
// rapidly so you should set the PG_TEMP environment variable to override
// the default location
type postgresEmbedded struct {
	kill      chan bool // Kill signal channel
	completed chan bool // Completed channel
	done      chan bool // Done channel; signalled when kill is complete
	DBDir     string    // Directory for database
	invalid   bool      // Process has terminated
	Port      int       // Port number for server
}

// newPostgresEmbedded creates a new PostgresEmbedded instance
func newPostgresEmbedded(dbDir string) *postgresEmbedded {
	return &postgresEmbedded{
		kill:      make(chan bool),
		completed: make(chan bool),
		done:      make(chan bool),
		invalid:   false,
		Port:      0,
		DBDir:     dbDir,
	}
}

// InitializeNew initializes a new PostgreSQL DB at p.DBDir
func (p *postgresEmbedded) InitializeNew() error {
	// Initialize PostgreSQL in the temp directory
	logging.Info("Initializing new database in: %s", p.DBDir)
	cmd := exec.Command("pg_ctl", "init", "-D", p.DBDir)
	output, err := cmd.Output()
	if err != nil {
		logging.Error("Unable to initialise database: %v -- %s", err, string(output))
		p.invalid = true
		return err
	}
	return nil
}

// GetConnectionOptions returns a string with connection options
func (p *postgresEmbedded) GetConnectionOptions() string {
	//return fmt.Sprintf("postgresql://127.0.0.1:%d/postgres?sslmode=disable", p.Port)
	return fmt.Sprintf("port=%d dbname=postgres sslmode=disable", p.Port)
}

// Start waiting for input
func (p *postgresEmbedded) startWaitingForInput(reader io.Reader, message string, found chan bool) {
	bytes := make([]byte, 1024)
	alltext := ""
	for {
		if count, err := reader.Read(bytes); err == nil && count > 0 {
			alltext += string(bytes[0:count])
			if strings.Index(alltext, message) > 0 {
				found <- true
				return
			}
		} else if err == io.EOF {
			found <- false
			return
		}
	}
}

// Wait for kill signal or server shutdown. When the server is done the
// temp directory will be removed and the done channel is signaled.
func (p *postgresEmbedded) waitForShutdown(cmd *exec.Cmd) {
	select {
	case <-p.kill:
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			logging.Warning("Unable to signal PostgreSQL daemon: %v", err)
		}
		cmd.Process.Wait()
	case <-p.completed:
		// OK - process is completed
	}
	p.removeTempDir()
	p.done <- true
}

// Clean up the temp dir
func (p *postgresEmbedded) removeTempDir() {
	if err := exec.Command("rm", "-fR", p.DBDir).Run(); err != nil {
		logging.Warning("Unable to remove temp dir for PostgreSQL instance: %v", err)
	}
}

// Stop terminates the embedded PostgreSQL process. The method will return when the
// temp directory is cleaned up.
func (p *postgresEmbedded) Stop() {
	if p.invalid {
		return
	}
	p.kill <- true
	select {
	case <-p.done:
		// OK - everything is done
	case <-time.After(1000 * time.Millisecond):
		// OK - timeout
	}
}

// CheckPostgresInstallation checks that PostgreSQL is installed. If it is missing it can't be launched.
func checkPostgresInstallation() bool {
	cmd := exec.Command("postgres", "--help")

	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// IsDatabaseDir naively checks for the existence of database directory.
func isDatabaseDir(databaseDir string) bool {
	if _, err := os.Stat(databaseDir); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// GetDBTempDir creates a temporary loradb folder at the default location
func getDBTempDir() (string, error) {
	databaseDir, err := ioutil.TempDir("", "loradb")
	if err != nil {
		logging.Error("Unable to create directory for PostgreSQL daemon: %v", err)
		return "", err
	}
	return databaseDir, nil
}

// Start launches the PostgreSQL server. If the given path contains an existing database instance, this instance will
// be opened. If not, a new instance will be created
func (p *postgresEmbedded) Start() error {
	port, err := utils.FreePort()
	if err != nil {
		p.invalid = true
		return err
	}
	p.Port = port
	// ...and launch postgresql
	cmd := exec.Command("postgres",
		"-r", filepath.Join(p.DBDir, "pg.txt"),
		"-D", p.DBDir,
		"-p", strconv.Itoa(p.Port))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logging.Error("Unable to get stdout pipe for PostgreSQL daemon: %v", err)
	}

	foundchan := make(chan bool)
	go p.startWaitingForInput(stderr,
		"ready to accept connections",
		foundchan)

	if err = cmd.Start(); err != nil {
		logging.Error("Unable to launch PostgreSQL daemon: %v", err)
		p.invalid = true
		return err
	}

	logging.Info("PostgreSQL data directory is at %s and daemon is running on port %d", p.DBDir, p.Port)

	// Signal the completed channel when the command is completed.
	go func() {
		cmd.Wait()
		p.completed <- true
	}()

	select {
	case found := <-foundchan:
		if !found {
			logging.Error("PostgreSQL daemon didn't report read")
		}
	case <-time.After(10 * time.Second):
		logging.Error("Timed ut waiting for ready log message from PostgreSQL daemon")
	}
	go p.waitForShutdown(cmd)
	return nil
}
