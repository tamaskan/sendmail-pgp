package main

import (
	"fmt"
	"io"
	"io/ioutil"

	"os"
	"strings"
	"time"

	parsemail "github.com/OfimaticSRL/parsemail"
	smtp "github.com/emersion/go-smtp"
	"github.com/n0madic/sendmail"
	log "github.com/sirupsen/logrus"
)

// The Backend implements SMTP server methods.
type Backend struct{}

func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after successful login.
type Session struct {
	From string
	To   []string
}

// AuthPlain check stub
func (s *Session) AuthPlain(username, password string) error {
	return nil
}

// Mail save sender
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	senderDomain := sendmail.GetDomainFromAddress(from)
	if len(senderDomains) > 0 && !senderDomains.Contains(senderDomain) {
		log.Errorf("Attempt to unauthorized send with domain %s", senderDomain)
		return fmt.Errorf("unauthorized sender domain %s", senderDomain)
	}
	s.From = from
	return nil
}

// Rcpt save recipients
func (s *Session) Rcpt(to string) error {
	s.To = strings.Split(to, ",")
	return nil
}

// Data receives the message body and sends it
func (s *Session) Data(r io.Reader) error {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	

	if (strings.Count(strings.Join(s.To,""), "@") > 1){
		log.Fatal("Multiple Recipients detected")
		log.Debug(strings.Join(s.To,""))
	}

	var pgprecipients = string(s.To[0])

	var stringhash = hasher(pgprecipients)

	email, err := parsemail.Parse(r) 
	if err != nil {
		log.Fatal("ohoh")
	}
	
	fmt.Println(email.Subject)
	fmt.Println(email.HTMLBody)

	pgpdata, err := os.ReadFile("/keys/"+stringhash+".pgp")
	
	if os.IsNotExist(err) {
		log.Debug("no /keys/"+stringhash+".pgp found, skipping encryption")
	} else {
		subject = "..."
		configdata, err := ioutil.ReadFile("/keys/"+stringhash+".config")
		if os.IsNotExist(err) {
			log.Debug("Config file /keys/"+stringhash+".config not found, encrypting everything")
			body = encrypter(pgpdata,[]byte(body))
		}else{
			if strings.Contains(string(configdata), email.Subject){
				
				var corepgp = string(encrypter(pgpdata,[]byte(body)))
				body = encrypter(pgpdata,[]byte(corepgp))

			}
		}
	}

	envelope, err := sendmail.NewEnvelope(&sendmail.Config{
		Sender:     s.From,
		Recipients: s.To,
		Subject:    subject,
		Body:       body,
	})
	if err != nil {
		return err
	}
	envelope.Send()
	errs := envelope.Send()
	for result := range errs {
		switch {
		case result.Level > sendmail.WarnLevel:
			log.WithFields(getLogFields(result.Fields)).Info(result.Message)
		case result.Level == sendmail.WarnLevel:
			log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
		case result.Level < sendmail.WarnLevel:
			log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
			return result.Error
		}
	}
	return nil
}

// Reset session
func (s *Session) Reset() {}

// Logout session
func (s *Session) Logout() error {
	return nil
}

// Start SMTP server
func startSMTP(bindAddr string) {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = bindAddr
	s.Domain = "sendmail"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	log.Info("Starting SMTP server at ", s.Addr)
	log.Fatal(s.ListenAndServe())
}
