package ddm

import (
	"fmt"
	"hash"
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

// NewHash returns a newly instantiated hashing function.
type NewHash func() hash.Hash

// DIBuilder incrementally builds the DDM Declaration Items structure for later serializing.
type DIBuilder struct {
	hash hash.Hash
	DeclarationItems
}

// NewDIBuilder constructs a new Declaration Items builder.
// It will panic if provided with a nil hasher.
func NewDIBuilder(newHash NewHash) *DIBuilder {
	if newHash == nil {
		panic("nil hasher")
	}
	hash := newHash()
	if hash == nil {
		panic("nil hash")
	}
	return &DIBuilder{
		hash: hash,
		DeclarationItems: DeclarationItems{
			Declarations: ManifestDeclarationItems{
				// init slices so they're non-nil for the JSON encoder.
				// Apple docs state they're required fields.
				Activations:    []ManifestDeclaration{},
				Assets:         []ManifestDeclaration{},
				Configurations: []ManifestDeclaration{},
				Management:     []ManifestDeclaration{},
			},
		},
	}
}

func tokenHashWrite(h hash.Hash, d *Declaration) {
	h.Write([]byte(d.ServerToken))
}

// Add adds a declaration d to the Declaration Items builder.
func (b *DIBuilder) Add(d *Declaration) {
	md := ManifestDeclaration{
		Identifier:  d.Identifier,
		ServerToken: d.ServerToken,
	}
	switch ManifestType(d.Type) {
	case "activation":
		b.Declarations.Activations = append(b.Declarations.Activations, md)
	case "asset":
		b.Declarations.Assets = append(b.Declarations.Assets, md)
	case "configuration":
		b.Declarations.Configurations = append(b.Declarations.Configurations, md)
	case "management":
		b.Declarations.Management = append(b.Declarations.Management, md)
	}
	tokenHashWrite(b.hash, d)
}

func tokenHashFinalize(h hash.Hash) string {
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Finalize finishes building the declarations items by computing the final Declarations Token.
func (b *DIBuilder) Finalize() {
	b.DeclarationItems.DeclarationsToken = tokenHashFinalize(b.hash)
}
