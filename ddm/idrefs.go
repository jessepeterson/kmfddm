package ddm

// IdentifierRefs is a map of declaration type to payload key paths.
// These key paths contain the identifiers of dependent declarations.
var IdentifierRefs = map[string][][]string{
	"com.apple.activation.simple": {
		{"StandardConfigurations"},
	},
	"com.apple.configuration.account.caldav": {
		{"AuthenticationCredentialsAssetReference"},
	},
	"com.apple.configuration.account.carddav": {
		{"AuthenticationCredentialsAssetReference"},
	},
	"com.apple.configuration.account.exchange": {
		{"UserIdentityAssetReference"},
		{"AuthenticationCredentialsAssetReference"},
	},
	"com.apple.configuration.account.google": {
		{"UserIdentityAssetReference"},
	},
	"com.apple.configuration.account.ldap": {
		{"AuthenticationCredentialsAssetReference"},
	},
	"com.apple.configuration.account.mail": {
		{"UserIdentityAssetReference"},
		{"IncomingServer", "AuthenticationCredentialsAssetReference"},
		{"OutgoingServer", "AuthenticationCredentialsAssetReference"},
	},
	"com.apple.configuration.account.subscribed-calendar": {
		{"AuthenticationCredentialsAssetReference"},
	},
}
