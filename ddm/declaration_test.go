package ddm

import (
	"testing"
)

const declTest1 = `{
	"Identifier": "com.example.test",
	"Type": "com.apple.test",
	"Payload": {}
}`

const declTest2 = `{
	"Identifier": "com.example.test",
	"Type": "com.apple.test",
	"Payload": {"foo": "bar"}
}`

const declActTest1 = `{
	"Identifier": "49F6F16A-70EB-4A89-B092-465FAEC5E550",
	"Payload": {
	  "StandardConfigurations": [
		"85B5130A-4D0D-462B-AA0D-0C3B6630E5AA",
		"0FCD2F56-D5BC-48EA-B98D-E0CCC0C6F9E0",
		"4D6F8451-C089-4E65-A615-7C6EFF154F72"
	  ]
	},
	"ServerToken": "8f700d70-f2d6-5b02-926d-ff36b5e47260",
	"Type": "com.apple.activation.simple"
  }`

const declMailTest1 = `{
	"Identifier": "49F6F16A-70EB-4A89-B092-465FAEC5E550",
	"Payload": {
		"IncomingServer": {
			"AuthenticationCredentialsAssetReference": "B962F496-0982-43D3-A203-CDF6FD5926F4"
		}
	},
	"ServerToken": "8f700d70-f2d6-5b02-926d-ff36b5e47260",
	"Type": "com.apple.configuration.account.mail"
  }`

func TestUnmarshal(t *testing.T) {
	d, err := ParseDeclaration([]byte(declTest1))
	if err != nil {
		t.Fatal(err)
	}
	if d.Identifier != "com.example.test" {
		t.Error("identifier mismatch")
	}
	if ManifestType(d.Type) != "test" {
		t.Error("type mismatch")
	}
}

func TestUnmarshalPayload(t *testing.T) {
	d, err := ParseDeclaration([]byte(declTest2))
	if err != nil {
		t.Fatal(err)
	}
	if d.Identifier != "com.example.test" {
		t.Error("identifier mismatch")
	}
	if ManifestType(d.Type) != "test" {
		t.Error("type mismatch")
	}
}

func TestIDRefs(t *testing.T) {
	d, err := ParseDeclaration([]byte(declActTest1))
	if err != nil {
		t.Fatal(err)
	}
	if ManifestType(d.Type) != "activation" {
		t.Error("type mismatch")
	}
	ids := d.IdentifierRefs
	for _, id := range []string{
		"85B5130A-4D0D-462B-AA0D-0C3B6630E5AA",
		"0FCD2F56-D5BC-48EA-B98D-E0CCC0C6F9E0",
		"4D6F8451-C089-4E65-A615-7C6EFF154F72",
	} {
		found := false
		for _, idDecl := range ids {
			if id == idDecl {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("could not find expected id %s in declaration IDRefs", id)
		}
	}
}

func TestIDRefs2(t *testing.T) {
	d, err := ParseDeclaration([]byte(declMailTest1))
	if err != nil {
		t.Fatal(err)
	}
	if ManifestType(d.Type) != "configuration" {
		t.Error("type mismatch")
	}
	ids := d.IdentifierRefs
	for _, id := range []string{
		"B962F496-0982-43D3-A203-CDF6FD5926F4",
	} {
		found := false
		for _, idDecl := range ids {
			if id == idDecl {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("could not find expected id %s in declaration IDRefs", id)
		}
	}
}
