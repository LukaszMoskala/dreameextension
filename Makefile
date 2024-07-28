all: dreameextension_linux_arm64 dreameextension_linux_arm
.PHONY: dreameextension_linux_arm64 dreameextension_linux_arm

dreameextension_linux_arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dreameextension_linux_arm64

dreameextension_linux_arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dreameextension_linux_arm