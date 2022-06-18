package main

import (
	"os"

	"github.com/devon-mar/dnsupdater/config"
	"github.com/devon-mar/dnsupdater/updater"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app        = kingpin.New("dnsupdater", "Insert DNS records from a file.")
	configFile = app.Flag("config", "Path to the config file.").Default("records.yml").String()
	checkCmd   = app.Command("check", "Check the config file.")
	insertCmd  = app.Command("insert", "Insert records.")
	batchSize  = insertCmd.Flag("batch", "Insert records in updates of the given size instead of per name.").Int()
)

func getUpdater(c *config.Config) updater.Updater {
	u := updater.NewRFC2136(c.Servers)
	if c.GSS != nil {
		if err := u.WithGSS(); err != nil {
			log.WithError(err).Error("error initializing GSS")
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
		log.WithError(err).Fatal("Error loading config")
	}
	u := getUpdater(c)
	defer u.Close()

	switch cmd {
	case checkCmd.FullCommand():
		log.Infof("Config is valid.")
	case insertCmd.FullCommand():
		if *batchSize != 0 {
			insertBatch(u, c.Zones, *batchSize)
		} else {
			insert(u, c.Zones)
		}
	}
}

func insert(s updater.Updater, zones map[string]*config.Zone) {
	for zoneName, zone := range zones {
		log.Infof("Inserting records for zone %q", zoneName)
		for _, r := range zone.Records {
			logger := log.WithFields(log.Fields{"name": r.Name, "zone": zoneName})
			insertRecords(s, zoneName, r.Records(zoneName), logger)
		}
	}
}

func insertBatch(s updater.Updater, zones map[string]*config.Zone, batchSize int) {
	for zoneName, zone := range zones {
		logger := log.WithField("zone", zoneName)
		logger.Infof("Insering records")
		var queue []dns.RR
		for _, r := range zone.Records {
			queue = append(queue, r.Records(zoneName)...)

			for len(queue) >= batchSize {
				insertRecords(s, zoneName, queue[:batchSize], logger)
				queue = queue[batchSize:]
			}
		}
		if len(queue) > 0 {
			insertRecords(s, zoneName, queue, logger)
		}
	}
}

func insertRecords(s updater.Updater, zone string, records []dns.RR, logger *log.Entry) {
	if err := s.Insert(dns.Fqdn(zone), records); err != nil {
		logger.WithError(err).Error("Error inserting records")
	}
}
