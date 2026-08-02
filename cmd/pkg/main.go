package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kuetix/std-mysql/modules"

	"github.com/kuetix/engine"
	"github.com/kuetix/engine/engine/domain"
	engineModule "github.com/kuetix/engine/modules"
)

var Version string
var BuildTime string

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <workflow_name> -[v|verbose]\n", os.Args[0])
		os.Exit(1)
	}

	workflow := os.Args[1]
	os.Args = os.Args[1:]
	verbose := flag.Bool("verbose", false, "Verbose mode")
	vFlag := flag.Bool("v", false, "Verbose mode")
	flag.Parse()

	engineModule.Enable()
	modules.Enable()

	verboseMode := *verbose || *vFlag

	response := engine.RunWorkflow("production", &domain.Options{
		Version:       Version,
		BuildTime:     BuildTime,
		EngineName:    "mysql-cli",
		ConfigName:    "engine",
		Workflow:      workflow,
		Verbose:       verboseMode,
		Quiet:         false,
		Amount:        1,
		Retry:         1,
		RetryDelay:    0,
		RestartPolicy: "stop",
		LogPath:       "stdout",
		Args:          flag.Args(),
		Context:       nil,
		Settings:      nil,
	})

	for _, res := range response {
		if res.Error != nil {
			fmt.Printf("Error: %s\n", res.Error)
			os.Exit(1)
		}
		if res.Response != nil {
			fmt.Printf("Result: %v\n", res.Response)
		}
	}
}
