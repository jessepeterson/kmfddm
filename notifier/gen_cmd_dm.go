package notifier

// git clone git@github.com:apple/device-management.git
// go install github.com/jessepeterson/admgen/cmd/admgencmd/...@latest

//go:generate admgencmd -pkg notifier -o cmd_dm.go -no-shared -no-responses -no-depend device-management/mdm/commands/declarativemanagement.yaml
