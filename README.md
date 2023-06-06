Combines n0madic/sendmail and ProtonMail/gopenpgp

go build in sendmail/cmd-folder

needs a /keys-folder with a [md5-hash-email].pgp and a [md5-hash-email].config file that contains the wording of the subject to enable encryption

needs eg.

export SENDMAIL_SMART_HOST=smtp.gmail.com:587

export SENDMAIL_SMART_LOGIN=test@gmail.com

export SENDMAIL_SMART_PASSWORD=password
