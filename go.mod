module github.com/jenish-rudani/HID_NFC_READER

go 1.21.3

require (
	bitbucket.org/bluvision-cloud/kit v1.0.64
	bitbucket.org/bluvision/pcsc v0.0.1
	github.com/ebfe/scard v0.0.0-20230420082256-7db3f9b7c8a7
	github.com/sirupsen/logrus v1.9.3
)

require golang.org/x/sys v0.25.0 // indirect

replace bitbucket.org/bluvision/pcsc => /Users/atom/Documents/BB/pcsc
