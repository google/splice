/*
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Binary service_windows implements the SpliceD Windows service.

When installed as a Window service, this daemon will wait for new domain
join requests, attempt the join, and return the output to the cloud datastore.

For debugging, run from an admin command shell with the 'debug' argument.
*/
package main

import (
	"golang.org/x/net/context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/deck/backends/eventlog"
	"github.com/google/deck/backends/logger"
	"github.com/google/deck"
	"golang.org/x/sys/windows/svc/debug"
	sysevt "golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows"
	"github.com/google/splice/generators"

	// register generators
	_ "github.com/google/splice/generators/prefix"
)

const svcName = "SpliceD"

var eventID = eventlog.EventID

// Type winSvc implements svc.Handler
type winSvc struct{}

// Execute starts the internal goroutine and waits for service signals from Windows.
func (m *winSvc) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	ctx := context.Background()
	errch := make(chan ExitEvt)

	changes <- svc.Status{State: svc.StartPending}
	if err := Init(); err != nil {
		deck.ErrorfA("Failure starting service. %v", err).With(eventID(EvtErrStartup)).Go()
		return
	}
	go func() {
		errch <- Run(ctx)
	}()
	deck.InfoA("Service started.").With(eventID(EvtStartup)).Go()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		// Watch for the spliced goroutine to fail for some reason.
		case err := <-errch:
			deck.ErrorA(err.Message).With(eventID(err.Code)).Go()
			break loop
		// Watch for service signals.
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				deck.ErrorfA("Unexpected control request #%d", c).With(eventID(EvtErrMisc)).Go()
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	var err error

	if isDebug {
		deck.Add(logger.Init(os.Stdout, 0))
	}

	deck.InfofA("Starting %s service.", name).With(eventID(EvtStartup)).Go()
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &winSvc{})
	if err != nil {
		deck.ErrorfA("%s service failed. %v", name, err).With(eventID(EvtErrMisc)).Go()
		return
	}
	deck.InfofA("%s service stopped.", name).With(eventID(EvtShutdown)).Go()
}

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       configure|debug.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	evt, err := eventlog.Init(svcName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	deck.Add(evt)
	defer deck.Close()

	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if isSvc {
		runService(svcName, false)
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		runService(svcName, true)
	case "configure":
		if err := Update(os.Args[2:]); err != nil {
			log.Fatalf("failed to configure application due to %v", err)
		}
		// Create the event source prior to opening to avoid description cannot be found errors.
		if err = sysevt.InstallAsEventCreate(svcName, sysevt.Info|sysevt.Warning|sysevt.Error); err != nil {
			if !strings.Contains(err.Error(), "registry key already exists") && err != windows.ERROR_ACCESS_DENIED {
				log.Fatalf("Installation of the event source returned %v", err)
				return
			}
		}
		fmt.Println("configuration successful")
	case "generators":
		fmt.Println("Available generators:")
		for _, v := range generators.List() {
			fmt.Printf("\t%s\n", v)
		}
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
}
