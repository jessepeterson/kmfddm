package ddm

import (
	"fmt"
	"hash"
	"strings"
)

// See https://developer.apple.com/documentation/devicemanagement/manifestdeclaration
type ManifestDeclaration struct {
	Identifier  string
	ServerToken string
}

// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse/manifestdeclarationitems
type ManifestDeclarationItems struct {
	Activations    []ManifestDeclaration
	Assets         []ManifestDeclaration
	Configurations []ManifestDeclaration
	Management     []ManifestDeclaration
}

// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
type DeclarationItems struct {
	Declarations      ManifestDeclarationItems
	DeclarationsToken string
}

// ManifestType returns the "type" of manifest from a declaration type.
// The result should (but is not guaranteed to) be one of "activation",
// "configuration", "asset", or "management".
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

type DIBuilder struct {
	DeclarationItems
	hash.Hash
}

func NewDIBuilder(newHash func() hash.Hash) *DIBuilder {
	b := &DIBuilder{
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
	if newHash != nil {
		b.Hash = newHash()
	}
	return b
}

func tokenHashWrite(h hash.Hash, d *Declaration) {
	if h == nil {
		return
	}
	h.Write([]byte(d.ServerToken))
}

func (b *DIBuilder) AddDeclarationData(d *Declaration) {
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
	tokenHashWrite(b.Hash, d)
}

func (b *DIBuilder) Finalize() {
	if b.Hash != nil {
		b.DeclarationItems.DeclarationsToken = fmt.Sprintf("%x", b.Hash.Sum(nil))
	}
}
