// Standalone drop-in replacement for sendmail with direct send
package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/n0madic/sendmail"
	log "github.com/sirupsen/logrus"
)

type arrayDomains []string

func (d *arrayDomains) String() string {
	return strings.Join(*d, ",")
}

func (d *arrayDomains) Set(value string) error {
	*d = append(*d, value)
	return nil
}

func (d *arrayDomains) Contains(str string) bool {
	for _, domain := range *d {
		if domain == str {
			return true
		}
	}
	return false
}

var (
	httpMode      bool
	httpBind      string
	httpToken     string
	ignored       bool
	ignoreDot     bool
	sender        string
	senderDomains arrayDomains
	smtpMode      bool
	smtpBind      string
	subject       string
	verbose       bool
)

func main() {
	flag.BoolVar(&ignored, "t", true, "Extract recipients from message headers. IGNORED")
	flag.BoolVar(&ignoreDot, "i", false, "When reading a message from standard input, don't treat a line with only a . character as the end of input.")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging for debugging purposes.")
	flag.StringVar(&sender, "f", "", "Set the envelope sender address.")
	flag.StringVar(&subject, "s", "", "Specify subject on command line.")

	flag.BoolVar(&httpMode, "http", false, "Enable HTTP server mode.")
	flag.StringVar(&httpBind, "httpBind", "localhost:8080", "TCP address to HTTP listen on.")
	flag.StringVar(&httpToken, "httpToken", "", "Use authorization token to receive mail (Token: header).")
	flag.BoolVar(&smtpMode, "smtp", false, "Enable SMTP server mode.")
	flag.StringVar(&smtpBind, "smtpBind", "localhost:25", "TCP or Unix address to SMTP listen on.")
	flag.Var(&senderDomains, "senderDomain", "Domain of the sender from which mail is allowed (otherwise all domains). Can be repeated many times.")

	flag.Parse()

	if !verbose {
		log.SetLevel(log.WarnLevel)
	}

	if httpMode || smtpMode {
		if httpMode {
			go startHTTP(httpBind)
		}
		if smtpMode {
			go startSMTP(smtpBind)
		}
		select {}
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			log.Fatal("no stdin input")
		}

		var body []byte
		bio := bufio.NewReader(os.Stdin)
		for {
			line, err := bio.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			if !ignoreDot && bytes.Equal(bytes.Trim(line, "\n"), []byte(".")) {
				break
			}
			body = append(body, line...)

		}

		if (strings.Count(strings.Join(flag.Args(),""), "@") > 1){
			log.Debug("Multiple Recipients detected")
			log.Debug(strings.Join(flag.Args(),""))
		}

		var pgprecipients = string(flag.Args()[0])

		var stringhash = hasher(pgprecipients)

		pgpdata, err := os.ReadFile("/keys/"+stringhash+".pgp")
		
		if os.IsNotExist(err) {
			print("no /keys/"+stringhash+".pgp found, skipping encryption")
		} else {
			subject = "..."
			configdata, err := ioutil.ReadFile("/keys/"+stringhash+".config")
			if os.IsNotExist(err) {
				print("Config file /keys/"+stringhash+".config not found, encrypting everything\r\n")
				body = encrypter(pgpdata,body)
			}else{
				if strings.Contains(string(configdata), subject){
					body = encrypter(pgpdata,body)
					print("Subject found, encrypting\r\n")
				}
			}
		}

		if len(body) == 0 {
			log.Fatal("Empty message body")
		}

		envelope, err := sendmail.NewEnvelope(&sendmail.Config{
			Sender:     os.Getenv("SENDMAIL_SMART_LOGIN"),
			Recipients: flag.Args(),
			Subject:    subject,
			Body:       body,
		})

		if err != nil {
			log.Fatal(err)
		}

		senderDomain := sendmail.GetDomainFromAddress(envelope.Header["From"][0])
		if len(senderDomains) > 0 && !senderDomains.Contains(senderDomain) {
			log.Fatalf("Attempt to unauthorized send with domain %s", senderDomain)
		}

		errs := envelope.Send()
		for result := range errs {
			switch {
			case result.Level > sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Info(result.Message)
			case result.Level == sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
			case result.Level < sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Fatal(result.Error)
			}
		}
	}
}

func hasher(pgprecipients string) string{
	hasher := md5.New()
    hasher.Write([]byte(pgprecipients))
	stringhash := hex.EncodeToString(hasher.Sum(nil))
	return stringhash
}

func encrypter(pgpdata []byte,body []byte) []byte{
	
	var sender = hasher(os.Getenv("SENDMAIL_SMART_LOGIN"))

	privkey, err := os.ReadFile("/keys/"+sender+".privpgp")
	if os.IsNotExist(err) {
		print("no /keys/"+sender+".privpgp found, skipping signing\r\n")
		armor, err := helper.EncryptMessageArmored(string(pgpdata), string(body))
		if(err != nil){log.Fatal(err)}
		log.Debug(armor)
		body = []byte(armor)
		return body
	} else {
		print("/keys/"+sender+".privpgp found, signing message\r\n")
		armor, err := helper.EncryptSignMessageArmored(string(pgpdata),string(privkey),[]byte(os.Getenv("SENDMAIL_SECRET")), string(body))
		if(err != nil){log.Fatal(err)}
		log.Debug(armor)
		body = []byte(armor)
		return body
	}
	
}

func getLogFields(fields sendmail.Fields) log.Fields {
	logFields := log.Fields{}
	if verbose {
		for k, v := range fields {
			logFields[k] = v
		}
	}
	return logFields
}
