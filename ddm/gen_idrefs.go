package ddm

// git clone git@github.com:apple/device-management.git
// go install github.com/jessepeterson/admgen/cmd/admgenddmrefs/...@latest

//go:generate admgenddmrefs -pkg ddm -name IdentifierRefs -o idrefs.go device-management/declarative/declarations/configurations
