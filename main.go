package main

import (
	"log/slog"
	"os"

	"github.com/devon-mar/dnsupdater/config"
	"github.com/devon-mar/dnsupdater/updater"

	"github.com/miekg/dns"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app        = kingpin.New("dnsupdater", "Insert DNS records from a file.")
	configFile = app.Flag("config", "Path to the config file.").Default("records.yml").String()
	checkCmd   = app.Command("check", "Check the config file.")
	insertCmd  = app.Command("insert", "Insert records.")
	batchSize  = insertCmd.Flag("batch", "Insert records in updates of the given size instead of per name.").Int()
	exitError  = insertCmd.Flag("exit-error", "Stop on the first error when inserting records.").Bool()
)

func getUpdater(c *config.Config) updater.Updater {
	u := updater.NewRFC2136(c.Servers)
	if c.GSS != nil {
		if err := u.WithGSS(); err != nil {
			slog.Error("error initializing GSS", "err", err)
			os.Exit(1)
		}
		if c.GSS.Username != "" {
			// c.Validate() already made sure that the reset of the fields are not empty
			u.WithCredentials(c.GSS.Username, c.GSS.Password, c.GSS.Domain)
		}
	}
	return u
}

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	c, err := config.ReadConfig(*configFile)
	if err != nil {
		slog.Error("Error loading config", "err", err)
		os.Exit(1)
	}
	u := getUpdater(c)
	defer u.Close()

	switch cmd {
	case checkCmd.FullCommand():
		slog.Info("Config is valid.")
	case insertCmd.FullCommand():
		if *batchSize != 0 {
			exit(insertBatch(u, c.Zones, *batchSize))
		} else {
			exit(insert(u, c.Zones))
		}
	}
}

func insert(s updater.Updater, zones map[string]*config.Zone) int {
	var ret int
	for zoneName, zone := range zones {
		slog.Info("Inserting records", "zone", zoneName)
		for _, r := range zone.Records {
			logger := slog.With("fqdn", r.FQDN, "zone", zoneName)
			ret += insertRecords(s, zoneName, r.Records(), logger)
		}
	}
	return ret
}

func insertBatch(s updater.Updater, zones map[string]*config.Zone, batchSize int) int {
	var ret int
	for zoneName, zone := range zones {
		logger := slog.With("zone", zoneName)
		logger.Info("Insering records")
		var queue []dns.RR
		for _, r := range zone.Records {
			queue = append(queue, r.Records()...)

			for len(queue) >= batchSize {
				ret += insertRecords(s, zoneName, queue[:batchSize], logger)
				queue = queue[batchSize:]
			}
		}
		if len(queue) > 0 {
			ret += insertRecords(s, zoneName, queue, logger)
		}
	}
	return ret
}

// if continueOnError is true, os.Exit(1) will be called.
func insertRecords(s updater.Updater, zone string, records []dns.RR, logger *slog.Logger) int {
	if err := s.Insert(dns.Fqdn(zone), records); err != nil {
		logger.Error("Error inserting records", "err", err)
		if *exitError {
			os.Exit(1)
		}
		return 1
	}
	return 0
}

// Exit, limiting the code to a max of 125 (as recommended by os.Exit).
func exit(code int) {
	if code > 125 {
		code = 125
	}
	os.Exit(code)
}
