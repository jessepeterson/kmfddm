package status

//go:generate declgen -repo https://github.com/apple/device-management.git -commit release -pkg status -out status.go -repl repls.yaml -path declarative/status -path declarative/protocol/statusreport.yaml
