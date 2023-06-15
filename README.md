Combines n0madic/sendmail and ProtonMail/gopenpgp

go build in sendmail/cmd-folder

needs a /keys-folder with a [md5-hash-email].pgp and optionally a [md5-hash-email].config file that contains the wording of the subject to enable encryption

needs docker env eg.

SENDMAIL_SMART_HOST=smtp.gmail.com

SENDMAIL_SMART_PORT=587

SENDMAIL_SMART_LOGIN=test@gmail.com

SENDMAIL_SMART_PASSWORD=password

SENDMAIL_SECRET=the password of your private pgp-file (if you don't want to expose it, change line 187 in cmd\sendmail\main.go)

if /keys/[md5-hash-email].privpgp of the sender exists, messages are signed and encrypted with obfuscated subject
