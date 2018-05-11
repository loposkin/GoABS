package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sealuzh/goabs/coverage/static"
	"github.com/sealuzh/goabs/deps"
	"github.com/sealuzh/goabs/utils/fsutil"
)

const argSize = 1

var filePath string
var gopath string
var projectPath string
var excludeTests bool
var recursivePackages bool
var fetchDeps bool
var printLogs bool

var logger *log.Logger

func parseArgs() error {
	flag.BoolVar(&excludeTests, "exclude-tests", false, "Indicate if test files should be excluded")
	flag.BoolVar(&recursivePackages, "rec-pkgs", false, "Define if package should be traversed recurseively")
	argGoPath := flag.String("gopath", "", "Sets the GOPATH for the project under study")
	flag.BoolVar(&fetchDeps, "fetch-deps", false, "Indicate to fetch dependencies automatically")
	//argProjectPath := flag.String("proj", "", "Declares root package of project. If not provided, the first argument needs to be the root package")
	flag.BoolVar(&printLogs, "logs", false, "Print logging to stdout")

	flag.Parse()
	args := flag.Args()
	lenArgs := len(args)
	if lenArgs != argSize {
		return fmt.Errorf("Argument size invalid. Expected %d, but was %d", argSize, lenArgs)
	}

	filePath = filepath.Clean(args[0])
	spFp := strings.Split(filePath, "/")
	if len(spFp) < 3 {
		// invalid filepath
		return fmt.Errorf("Invalid filepath '%s'. Must start with 'hosting-provider/user/repo'", filePath)
	}
	projectPath = filepath.Join(spFp[0], spFp[1], spFp[2])

	if *argGoPath == "" {
		gopath = os.Getenv("GOPATH")
		if gopath == "" {
			return fmt.Errorf("GOPATH not set and not provided")
		}
	} else {
		argGoPathExpanded, err := fsutil.ExpandTilde(*argGoPath)
		if err != nil {
			return err
		}
		gopath = argGoPathExpanded
	}

	// projectPath = *argProjectPath
	// if projectPath != "" {
	// 	if len(strings.Split(projectPath, "/")) != 3 {
	// 		// project path is not root package
	// 		return fmt.Errorf("Project path ('%s') is not a root package. Must be of form 'hosting-provider/user/repo'", projectPath)
	// 	}
	// } else {
	// 	projectPath = filepath.Join(spFp[0], spFp[1], spFp[2])
	// }

	return nil
}

func main() {
	err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		flag.Usage()
		return
	}

	logger = log.New(os.Stdout, "# ", log.Ldate|log.Lmicroseconds|log.Llongfile|log.LUTC)
	if !printLogs {
		logger.SetOutput(ioutil.Discard)
	}
	printConfig()

	// fetch dependencies
	if fetchDeps {
		err := deps.Fetch(filepath.Join(gopath, "src", projectPath))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not fetch dependencies:\n %v\n\n", err)
		}
	}

	// start finding call sites
	f := static.NewStaticCallSiteFinder(filePath, gopath, recursivePackages, excludeTests)
	if f == nil {
		fmt.Fprintln(os.Stderr, "Got nil from static.NewStaticCallSiteFinder")
		return
	}

	err = f.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	css, err := f.All()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	for _, cs := range css {
		fmt.Fprintln(os.Stdout, cs)
	}
}

func printConfig() {
	logger.Printf("gopath: %s\n", gopath)
	logger.Printf("root package: %s\n", projectPath)
	logger.Printf("file path: %s\n", filePath)
}
