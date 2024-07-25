package build

import (
	"hash"

	"github.com/jessepeterson/kmfddm/ddm"
)

// DIBuilder incrementally builds the DDM Declaration Items structure for later serializing.
type DIBuilder struct {
	hash hash.Hash
	ddm.DeclarationItems
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
		DeclarationItems: ddm.DeclarationItems{
			Declarations: ddm.ManifestDeclarationItems{
				// init slices so they're non-nil for the JSON encoder.
				// Apple docs state they're required fields.
				Activations:    []ddm.ManifestDeclaration{},
				Assets:         []ddm.ManifestDeclaration{},
				Configurations: []ddm.ManifestDeclaration{},
				Management:     []ddm.ManifestDeclaration{},
			},
		},
	}
}

// Add adds a declaration d to the Declaration Items builder.
func (b *DIBuilder) Add(d *ddm.Declaration) {
	md := ddm.ManifestDeclaration{
		Identifier:  d.Identifier,
		ServerToken: d.ServerToken,
	}
	switch ddm.ManifestType(d.Type) {
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

// Finalize finishes building the declarations items by computing the final Declarations Token.
func (b *DIBuilder) Finalize() {
	b.DeclarationItems.DeclarationsToken = tokenHashFinalize(b.hash)
}
