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

/*Binary service_windows implements the SpliceD Windows service.

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

	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows"
)

const svcName = "SpliceD"

var elog debug.Log

// Type winSvc implements svc.Handler
type winSvc struct{}

// Execute starts the internal goroutine and waits for service signals from Windows.
func (m *winSvc) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	ctx := context.Background()
	errch := make(chan ExitEvt)

	changes <- svc.Status{State: svc.StartPending}
	if err := Init(); err != nil {
		elog.Error(EvtErrStartup, fmt.Sprintf("Failure starting service. %v", err))
		return
	}
	go func() {
		errch <- Run(ctx)
	}()
	elog.Info(EvtStartup, "Service started.")

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		// Watch for the spliced goroutine to fail for some reason.
		case err := <-errch:
			elog.Error(err.Code, err.Message)
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
				elog.Error(EvtErrMisc, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(EvtStartup, fmt.Sprintf("Starting %s service.", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &winSvc{})
	if err != nil {
		elog.Error(EvtErrMisc, fmt.Sprintf("%s service failed. %v", name, err))
		return
	}
	elog.Info(EvtShutdown, fmt.Sprintf("%s service stopped.", name))
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
		if err = eventlog.InstallAsEventCreate(svcName, eventlog.Info|eventlog.Warning|eventlog.Error); err != nil {
			if !strings.Contains(err.Error(), "registry key already exists") && err != windows.ERROR_ACCESS_DENIED {
				log.Fatalf("Installation of the event source returned %v", err)
				return
			}
		}
		fmt.Println("configuration successful")
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
}
