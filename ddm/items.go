package ddm

import (
	"strings"
)

// ManifestDeclaration contains the identifier and server token of a declaration.
// See https://developer.apple.com/documentation/devicemanagement/manifestdeclaration
type ManifestDeclaration struct {
	Identifier  string
	ServerToken string
}

// ManifestDeclarationItems contains a listing of manifest-type delineated declarations.
// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse/manifestdeclarationitems
type ManifestDeclarationItems struct {
	Activations    []ManifestDeclaration
	Assets         []ManifestDeclaration
	Configurations []ManifestDeclaration
	Management     []ManifestDeclaration
}

// DeclarationItems are the set of declartions a DDM client should synchronize.
// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
type DeclarationItems struct {
	Declarations      ManifestDeclarationItems
	DeclarationsToken string
}

// ManifestType returns the "type" of manifest from a declaration type.
// The result should (but is not guaranteed to) be one of "activation",
// "configuration", "asset", or "management".
//
// Generally this is the third word separated by periods in a
// declaration type. For example the manifest type of the declaration
// type "com.apple.configuration.management.test" should be
// "configuration".
func ManifestType(t string) string {
	if !strings.HasPrefix(t, "com.apple.") {
		return ""
	}
	t = t[10:]
	pos := strings.IndexByte(t, '.')
	if pos == -1 {
		return t
	}
	return t[0:pos]
}
