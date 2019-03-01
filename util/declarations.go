package util

import "time"

const (
	configDir  = "."
	configFile = configDir + "/config.json"
	reportFile = configDir + "/report.json"
	queueFile  = configDir + "/queue.json"
	logFile    = configDir + "/ispmon.log"
)

type (
	Configuration struct {
		LogLevel       string
		ConfigDir      string
		ReportTitle    string
		ISP            string
		Emails         ReportEmails
		ClientID       string
		CheckHost      string
		Frequency      int
		ReportOutage   int //Minimum outage duration, in minutes, that triggers an email to the ISP
		SendgridApiKey string
		Message        string
	}

	ReportEntry struct {
		Time   time.Time
		ID     string
		Status string //online/offline/slow
	}

	ReportEmails struct {
		From     string
		FromMail string
		ToMails  []string
		Bcc      []string
	}

	Report struct {
		Title   string
		Entries []ReportEntry
	}

	Email struct {
		From           string
		FromMail       string
		To             string
		ToMails        []string
		Bcc            []string
		Subject        string
		Plaintext      string
		HTML           string
		AttachmentPath string
	}
)
