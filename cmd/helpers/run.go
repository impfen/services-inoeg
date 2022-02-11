// Kiebitz - Privacy-Friendly Appointment Scheduling
// Copyright (C) 2021-2021 The Kiebitz Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version. Additional terms
// as defined in section 7 of the license (e.g. regarding attribution)
// are specified at https://kiebitz.eu/en/docs/open-source/additional-terms.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package helpers

import (
	"fmt"
	"github.com/impfen/services-inoeg"
	"github.com/impfen/services-inoeg/helpers"
	"github.com/urfave/cli"
	"os"
	"os/signal"
	"syscall"
)

func wait() {
	// we wait for CTRL-C / Interrupt
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	services.Log.Info("Waiting for CTRL-C...")
	<-sigchan
}

type Server interface {
	Start() error
	Stop() error
}

func waitAndStop(servers []Server) error {

	var lastErr error

	wait()

	for _, server := range servers {
		if err := server.Stop(); err != nil {
			lastErr = err
			services.Log.Error(err)
		}
	}

	return lastErr
}

func initializeStorage(settings *services.Settings) (Server, error) {
	services.Log.Debug("Starting storage server...")
	if settings.Storage == nil {
		return nil, fmt.Errorf("Storage settings undefined")
	}
	return helpers.InitializeStorageServer(settings)
}

func initializeAppointments(settings *services.Settings) (Server, error) {
	services.Log.Debug("Starting appointments server...")
	if settings.Appointments == nil {
		return nil, fmt.Errorf("Appointments settings undefined")
	}
	return helpers.InitializeAppointmentsServer(settings)
}

func initializeMetrics(settings *services.Settings) (Server, error) {
	services.Log.Debug("Starting metrics server...")
	if settings.Metrics == nil {
		return nil, nil
		return nil, fmt.Errorf("Metrics settings undefined")
	}
	return helpers.InitializeMetricsServer(settings)
}

type Initializer func(settings *services.Settings) (Server, error)

func startServer(settings *services.Settings, initializer Initializer) Server {

	server, err := initializer(settings)

	if err != nil {
		services.Log.Fatal(err)
	}

	if server == nil {
		return nil
	}

	if err := server.Start(); err != nil {
		services.Log.Fatal(err)
	}

	return server

}

func run(settings *services.Settings, initializers []Initializer) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		servers := make([]Server, 0)
		for _, initializer := range initializers {
			if server := startServer(settings, initializer); server != nil {
				servers = append(servers, server)
			}
		}
		return waitAndStop(servers)
	}
}

func Run(settings *services.Settings) ([]cli.Command, error) {

	return []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"s"},
			Flags:   []cli.Flag{},
			Usage:   "Run the different servers.",
			Subcommands: []cli.Command{
				{
					Name:   "all",
					Flags:  []cli.Flag{},
					Usage:  "Run all servers at once.",
					Action: run(settings, []Initializer{initializeMetrics, initializeStorage, initializeAppointments}),
				},
				{
					Name:   "storage",
					Flags:  []cli.Flag{},
					Usage:  "Run the storage server.",
					Action: run(settings, []Initializer{initializeMetrics, initializeStorage}),
				},
				{
					Name:   "appointments",
					Flags:  []cli.Flag{},
					Usage:  "Run the appointments server.",
					Action: run(settings, []Initializer{initializeMetrics, initializeAppointments}),
				},
			},
		},
	}, nil
}
