package main

import (
	"text/template"
	"bytes"
	"github.com/SlyMarbo/gmail"
	"os"
	"bufio"
	"strings"
	"io"
	"log"
	"io/ioutil"
	"encoding/json"
	"time"
	"errors"
)

type Recipient struct {
	Source string
	Fname string
	Email string
}

type Config struct {
	SMTPLogin string
	SMTPPassword string
	EmailSubject string
	Timeout int
}

var Log *log.Logger
var config Config

func ReadConfig()(Config) {
	file, e := ioutil.ReadFile("config.json")
	if e != nil { panic(e) }

	var conf Config
	e = json.Unmarshal(file, &conf)
	if e != nil { panic(e) }

	return conf
}

func InitLogger(infoHandle io.Writer) () {
	Log = log.New(infoHandle,"", log.Ldate|log.Ltime)
}

func ReadRecipients() ([]Recipient, error) {
	var ret[] Recipient
	filename := "recipients.txt"
	if len(filename) == 0 {
		return ret, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return ret, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var idata Recipient
	var split []string
	for {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		}

		if err != nil {
			return ret, err
		}

		line = strings.TrimSpace(line)

		// Skip commented lines
		if len(line) < 1 || line[0] == '#' {
			continue
		}

		split = strings.SplitN(line, "|", 2)

		// Skip lines with incomplete data
		if len(split) != 2 {
			continue
		}

		idata.Source = line
		idata.Fname = strings.TrimSpace(split[0])
		idata.Email = strings.TrimSpace(split[1])

		ret = append(ret, idata)
	}
	return ret, nil
}

func SendMail(recipient Recipient, subj string, body string) (error) {
	email := gmail.Compose(subj, body)
	email.From = config.SMTPLogin
	email.Password = config.SMTPPassword

	// Defaults to "text/plain; charset=utf-8" if unset.
	email.ContentType = "text/html; charset=utf-8"

	// Normally you'll only need one of these, but I thought I'd show both.
	email.AddRecipient(recipient.Email)

	err := email.Send()
	if err != nil {
		Log.Printf("Failed to send email to %s", recipient.Email)
		return errors.New("Send failed")
	}
	Log.Printf("Email successfully sent to %s", recipient.Email)
	return nil
}

func ExecuteTemplate(tmpl *template.Template, recipient Recipient) (string) {
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, recipient)
	if err != nil { panic(err) }
	ret := buf.String()

	return ret
}

func main() {
	logFile, lerr := os.OpenFile("mailer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if lerr != nil { panic(lerr) }

	InitLogger(logFile)

	config = ReadConfig()

	Log.Printf("---------- New mailer session started ----------\n")

	mail_tmpl, err := template.ParseFiles("mail_text")
	if err != nil { panic(err) }

	recipients, ferr := ReadRecipients()
	if ferr != nil { panic(err) }

	rlen := len(recipients)

	if rlen < 1 {
		panic("No data in recipients.txt")
	}

	subj_tmpl, err := template.New("mail_subj").Parse(config.EmailSubject)
	if err != nil { panic(err) }

	for i := 0; i < rlen; i++  {
		mailBody := ExecuteTemplate(mail_tmpl, recipients[i])
		mailSubj := ExecuteTemplate(subj_tmpl, recipients[i])

		err = SendMail(recipients[i], mailSubj, mailBody)

		if config.Timeout > 0 {
			pause := time.Duration(config.Timeout) * time.Second
			time.Sleep(pause)
		}
	}
}