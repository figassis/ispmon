package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/thanhpk/randstr"

	"github.com/gobuffalo/uuid"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

const (
	HOUR, DAY    = time.Hour, time.Hour * 24
	LETTER_BYTES = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	Config *Configuration
)

func LoadConfig() (err error) {
	jsonFile, err := os.Open(configFile)
	if err != nil {
		return
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(byteValue), &Config); err != nil {
		return
	}

	return
}

func loadReport(path string) (r Report) {
	jsonFile, err := os.Open(path)
	if err = Log(1, err); err != nil {
		return Report{Title: Config.ReportTitle}
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err = Log(1, err); err != nil {
		return Report{Title: Config.ReportTitle}
	}

	if err = Log(1, json.Unmarshal([]byte(byteValue), &r)); err != nil {
		return Report{Title: Config.ReportTitle}
	}

	return
}

func NewUUID() (result string) {
	newUuid, err := uuid.NewV4()
	if err = Log(1, err); err != nil {
		return ""
	}
	return newUuid.String()
}

func Random(n int) string {
	return randstr.String(n)
}

func RandomHex(n int) string {
	return randstr.Hex(n)
}

func httpGet(url string) (response []byte, err error) {
	client := http.Client{Timeout: time.Duration(5 * time.Second)}

	resp, err := client.Get(url)
	if err = Log(1, err); err != nil {
		return
	}

	defer resp.Body.Close()

	response, err = ioutil.ReadAll(resp.Body)
	if err = Log(1, err); err != nil {
		return []byte{}, err
	}

	if resp.StatusCode != 200 {
		err = errors.New("HttpGet failed with code " + fmt.Sprint(resp.StatusCode))
		return []byte{}, err
	}

	return
}

func Run(Wg *sync.WaitGroup, errs chan error) {
	defer Wg.Done()
	for {
		if err := Log(1, processAgent()); err != nil {
			select {
			case errs <- err:
				break
			default:
				Log(1, fmt.Sprint("Error channel is full"))
			}
		}
		time.Sleep(time.Minute * time.Duration(Config.Frequency))
	}
}

func readFile(path string) (content string, err error) {
	b, err := ioutil.ReadFile(path)
	if err = Log(1, err); err != nil {
		return
	}
	return string(b), nil
}

func writeFile(path, content string) (err error) {
	file, err := os.Create(path)
	if err = Log(1, err); err != nil {
		return
	}
	defer file.Close()

	fmt.Fprintf(file, content)
	return
}

func saveReport(r Report, path string) (err error) {

	var duration time.Duration
	if len(r.Entries) != 0 {
		duration = r.Entries[len(r.Entries)-1].Time.Sub(r.Entries[0].Time)
	}

	if duration > 0 {
		r.Title = fmt.Sprintf("%s - Offline for %s", Config.ReportTitle, duration.String())
	}

	obj, err := json.Marshal(r)
	if err = Log(1, err); err != nil {
		return
	}

	var out bytes.Buffer
	err = json.Indent(&out, obj, "", "\t")
	if err = Log(1, err); err != nil {
		return
	}

	return Log(1, writeFile(path, string(out.Bytes())))
}

func checkConnectivity() (err error) {
	report := loadReport(reportFile)
	queue := loadReport(queueFile)

	id := NewUUID()
	_, err = httpGet(Config.CheckHost)
	if err = Log(1, err); err != nil {
		report.Entries = append(report.Entries, ReportEntry{time.Now(), id, "offline"})
		queue.Entries = append(queue.Entries, ReportEntry{time.Now(), id, "offline"})
		Log(1, saveReport(queue, queueFile))
	} else {
		report.Entries = append(report.Entries, ReportEntry{time.Now(), id, "online"})
	}

	Log(1, saveReport(report, reportFile))
	return Log(1, err)
}

func clearReport(path string) (err error) {
	report := loadReport(path)

	report.Entries = []ReportEntry{}
	if err = Log(1, saveReport(report, path)); err != nil {
		return
	}
	return
}

func reportIssue() (err error) {
	queue := loadReport(queueFile)

	var duration time.Duration
	if len(queue.Entries) != 0 {
		duration = queue.Entries[len(queue.Entries)-1].Time.Sub(queue.Entries[0].Time)
	}

	if duration < time.Minute*time.Duration(Config.ReportOutage) {
		Log(1, clearReport(queueFile))
		//return Log(1, fmt.Sprintf("There was an outage of %s. No need to conact ISP", duration.String()))
		return nil
	}

	fileName := "./OutageReport-" + time.Now().Format("Mon Jan 2 15:04:05") + ".json"
	content, err := readFile(queueFile)
	if err = Log(1, err); err != nil {
		return
	}

	if err = Log(1, writeFile(fileName, content)); err != nil {
		return
	}

	defer os.Remove(fileName)

	email := Email{
		From:           Config.Emails.From,
		FromMail:       Config.Emails.FromMail,
		To:             Config.ISP,
		ToMails:        Config.Emails.ToMails,
		Subject:        Config.ReportTitle,
		Plaintext:      Config.Message,
		HTML:           Config.Message,
		AttachmentPath: fileName,
	}

	if _, err = email.Send(); err != nil {
		return Log(1, err)
	}

	Log(1, clearReport(queueFile))
	return
}

func processAgent() (err error) {
	if err = Log(1, checkConnectivity()); err != nil {
		return
	}

	//We now have a connection, check if there is something to report
	if err = Log(1, reportIssue()); err != nil {
		return
	}
	return
}

func (msg *Email) Send() (responseCode string, err error) {
	if Config.SendgridApiKey == "" {
		return "", Log(3, errors.New("Email API Key not configured"))
	}

	if len(msg.ToMails) == 0 || msg.To == "" {
		return "", Log(3, errors.New("Empty destination details"))
	}

	if msg.From == "" || msg.FromMail == "" {
		return "", Log(3, errors.New("Empty sender details"))
	}

	if msg.Subject == "" {
		return "", Log(3, errors.New("Empty subject"))
	}

	message := mail.NewV3Mail()
	message.From = mail.NewEmail(msg.From, msg.FromMail)
	message.Subject = msg.Subject
	message.AddContent(mail.NewContent("text/plain", msg.Plaintext), mail.NewContent("text/html", msg.HTML))

	personalization := mail.NewPersonalization()
	for _, to := range msg.ToMails {
		personalization.AddTos(mail.NewEmail("", to))
	}

	for _, bcc := range msg.Bcc {
		personalization.AddBCCs(mail.NewEmail("", bcc))
	}

	content, err := readFile(msg.AttachmentPath)
	if err = Log(1, err); err != nil {
		return
	}

	attachment := mail.Attachment{
		Content:     base64.StdEncoding.EncodeToString([]byte(content)),
		Type:        "application/json",
		Name:        strings.TrimPrefix(msg.AttachmentPath, "./"),
		Filename:    strings.TrimPrefix(msg.AttachmentPath, "./"),
		Disposition: "attachment",
		ContentID:   "Outage Report",
	}
	message.AddAttachment(&attachment)

	message.AddPersonalizations(personalization)

	client := sendgrid.NewSendClient(Config.SendgridApiKey)
	response, err := client.Send(message)
	if err = Log(3, err); err != nil {
		return "", Log(3, errors.New("Sending email error"))
	}

	Log(1, fmt.Sprintf("Email response %v", response))

	return fmt.Sprint(response.StatusCode), err
}
