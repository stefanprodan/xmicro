package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	unrender "github.com/unrolled/render"
)

// AppContext app-wide configuration
type AppContext struct {
	Role      string
	Version   string
	Env       string
	Hostname  string
	Port      int
	StartTime time.Time
	WorkDir   string
	Render    *unrender.Render `json:"-"`
}

var appCtx *AppContext

func initCtx(env string, port int, role string) error {

	version, err := parseVersionFile("VERSION")
	if err != nil {
		return err
	}
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	hostname, _ := os.Hostname()
	if err != nil {
		return err
	}
	appCtx = &AppContext{
		Render: unrender.New(unrender.Options{
			IndentJSON: true,
			Layout:     "layout",
		}),
		Version:   version,
		WorkDir:   workDir,
		Hostname:  hostname,
		Env:       env,
		Port:      port,
		Role:      role,
		StartTime: time.Now(),
	}

	return nil
}

func parseVersionFile(versionPath string) (string, error) {
	dat, err := ioutil.ReadFile(versionPath)
	if err != nil {
		return "", fmt.Errorf("error reading version file %s", err.Error())
	}
	version := string(dat)
	version = strings.Trim(strings.Trim(version, "\n"), " ")
	semverRegex := `^v?(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[\da-z\-]+(?:\.[\da-z\-]+)*)?(?:\+[\da-z\-]+(?:\.[\da-z\-]+)*)?$`
	match, err := regexp.MatchString(semverRegex, version)
	if err != nil {
		return "", fmt.Errorf("error executing version regex match %s", err.Error())
	}
	if !match {
		return "", fmt.Errorf("string in version file is not a valid version number")
	}

	return version, nil
}
