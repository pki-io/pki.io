package entity

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"github.com/pki-io/pki.io/crypto"
	"github.com/pki-io/pki.io/document"
)

const EntityDefault string = `{
    "scope": "pki.io",
    "version": 1,
    "type": "entity-document",
    "options": "",
    "body": {
      "id": "",
      "name": "",
      "key-type": "ec",
      "public-signing-key": "",
      "private-signing-key": "",
      "public-encryption-key": "",
      "private-encryption-key": ""
    }
}`

const EntitySchema string = `{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "EntityDocument",
  "description": "Entity Document",
  "type": "object",
  "required": ["scope","version","type","options","body"],
  "additionalProperties": false,
  "properties": {
      "scope": {
          "description": "Scope of the document",
          "type": "string"
      },
      "version": {
          "description": "Document schema version",
          "type": "integer"
      },
      "type": {
          "description": "Type of document",
          "type": "string"
      },
      "options": {
          "description": "Options data",
          "type": "string"
      },
      "body": {
          "description": "Body data",
          "type": "object",
          "required": ["id", "name", "key-type", "public-signing-key", "private-signing-key", "public-encryption-key", "private-encryption-key"],
          "additionalProperties": false,
          "properties": {
              "id" : {
                  "description": "Entity ID",
                  "type": "string"
              },
              "name" : {
                  "description": "Entity name",
                  "type": "string"
              },
              "key-type": {
				  "description": "Key type. Either rsa or ec",
				  "type": "string"
              },
              "public-signing-key" : {
                  "description": "Public signing key",
                  "type": "string"
              },
              "private-signing-key" : {
                  "description": "Private signing key",
                  "type": "string"
              },
              "public-encryption-key" : {
                  "description": "Public encryption key",
                  "type": "string"
              },
              "private-encryption-key" : {
                  "description": "Private encryption key",
                  "type": "string"
              }
          }
      }
  }
}`

type EntityData struct {
	Scope   string `json:"scope"`
	Version int    `json:"version"`
	Type    string `json:"type"`
	Options string `json:"options"`
	Body    struct {
		Id                   string `json:"id"`
		Name                 string `json:"name"`
		KeyType              string `json:"key-type"`
		PublicSigningKey     string `json:"public-signing-key"`
		PrivateSigningKey    string `json:"private-signing-key"`
		PublicEncryptionKey  string `json:"public-encryption-key"`
		PrivateEncryptionKey string `json:"private-encryption-key"`
	} `json:"body"`
}

type Entity struct {
	document.Document
	Data EntityData
}

func New(jsonString interface{}) (*Entity, error) {
	entity := new(Entity)
	if err := entity.New(jsonString); err != nil {
		return nil, fmt.Errorf("Couldn't create new entity: %s", err)
	} else {
		return entity, nil
	}
}

func (entity *Entity) New(jsonString interface{}) error {
	entity.Schema = EntitySchema
	entity.Default = EntityDefault
	if err := entity.Load(jsonString); err != nil {
		return fmt.Errorf("Could not create new Entity: %s", err)
	} else {
		return nil
	}
}

func (entity *Entity) Load(jsonString interface{}) error {
	data := new(EntityData)
	if data, err := entity.FromJson(jsonString, data); err != nil {
		return fmt.Errorf("Could not load entity JSON: %s", err)
	} else {
		entity.Data = *data.(*EntityData)
		return nil
	}
}

func (entity *Entity) Dump() string {
	if jsonString, err := entity.ToJson(entity.Data); err != nil {
		return ""
	} else {
		return jsonString
	}
}

func (entity *Entity) DumpPublic() string {
	public, err := entity.Public()
	if err != nil {
		return ""
	} else {
		return public.Dump()
	}
}

func (entity *Entity) generateRSAKeys() (*rsa.PrivateKey, *rsa.PrivateKey, error) {
	signingKey, err := crypto.GenerateRSAKey()
	if err != nil {
		return nil, nil, err
	}

	encryptionKey, err := crypto.GenerateRSAKey()
	if err != nil {
		return nil, nil, err
	}

	signingKey.Precompute()
	encryptionKey.Precompute()

	if err := signingKey.Validate(); err != nil {
		return nil, nil, fmt.Errorf("Could not validate signing key: %s", err)
	}

	if err := encryptionKey.Validate(); err != nil {
		return nil, nil, fmt.Errorf("Could not validate encryption key: %s", err)
	}

	if pub, err := crypto.PemEncodePublic(&signingKey.PublicKey); err != nil {
		return nil, nil, err
	} else {
		entity.Data.Body.PublicSigningKey = string(pub)
	}

	return signingKey, encryptionKey, nil
}

func (entity *Entity) generateECKeys() (*ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	signingKey, err := crypto.GenerateECKey()
	if err != nil {
		return nil, nil, err
	}

	encryptionKey, err := crypto.GenerateECKey()
	if err != nil {
		return nil, nil, err
	}

	// TODO: Do we need to do any validation here?

	return signingKey, encryptionKey, nil
}

func (entity *Entity) GenerateKeys() error {
	var signingKey interface{}
	var encryptionKey interface{}
	var publicSigningKey interface{}
	var publicEncryptionKey interface{}
	var err error
	switch crypto.KeyType(entity.Data.Body.KeyType) {
	case crypto.KeyTypeRSA:
		signingKey, encryptionKey, err = entity.generateRSAKeys()
		if err != nil {
			return err
		}
		publicSigningKey = &signingKey.(*rsa.PrivateKey).PublicKey
		publicEncryptionKey = &encryptionKey.(*rsa.PrivateKey).PublicKey
	case crypto.KeyTypeEC:
		signingKey, encryptionKey, err = entity.generateECKeys()
		if err != nil {
			return err
		}
		publicSigningKey = &signingKey.(*ecdsa.PrivateKey).PublicKey
		publicEncryptionKey = &encryptionKey.(*ecdsa.PrivateKey).PublicKey
	default:
		return fmt.Errorf("Invalid key type: %s", entity.Data.Body.KeyType)
	}

	if pub, err := crypto.PemEncodePublic(publicSigningKey); err != nil {
		return err
	} else {
		entity.Data.Body.PublicSigningKey = string(pub)
	}

	if key, err := crypto.PemEncodePrivate(signingKey); err != nil {
		return err
	} else {
		entity.Data.Body.PrivateSigningKey = string(key)
	}

	if pub, err := crypto.PemEncodePublic(publicEncryptionKey); err != nil {
		return err
	} else {
		entity.Data.Body.PublicEncryptionKey = string(pub)
	}

	if key, err := crypto.PemEncodePrivate(encryptionKey); err != nil {
		return err
	} else {
		entity.Data.Body.PrivateEncryptionKey = string(key)
	}

	return nil
}

func (entity *Entity) Sign(container *document.Container) error {
	var signatureMode crypto.Mode
	switch crypto.KeyType(entity.Data.Body.KeyType) {
	case crypto.KeyTypeRSA:
		signatureMode = crypto.SignatureModeSha256Rsa
	case crypto.KeyTypeEC:
		signatureMode = crypto.SignatureModeSha256Ecdsa
	default:
		return fmt.Errorf("Invalid key type: %s", entity.Data.Body.KeyType)
	}

	signature := crypto.NewSignature(signatureMode)
	container.Data.Options.SignatureMode = string(signature.Mode)
	// Force a clear of any existing signature values as that doesn't make sense
	container.Data.Options.Signature = ""

	containerJson := container.Dump()

	if err := crypto.Sign(containerJson, entity.Data.Body.PrivateSigningKey, signature); err != nil {
		return fmt.Errorf("Could not sign container json: %s", err)
	}
	if signature.Message != containerJson {
		return fmt.Errorf("Signed message doesn't match input")
	}

	container.Data.Options.SignatureMode = string(signature.Mode)
	container.Data.Options.Signature = signature.Signature
	return nil
}

func (entity *Entity) Authenticate(container *document.Container, id, key string) error {
	rawKey, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("Could not decode key: %s", err)
	}

	newKey, salt, err := crypto.ExpandKey(rawKey, nil)
	if err != nil {
		return fmt.Errorf("Cold not expand key: %s", err)
	}

	signature := crypto.NewSignature(crypto.SignatureModeSha256Hmac)
	container.Data.Options.SignatureMode = string(signature.Mode)
	signatureInputs := make(map[string]string)
	signatureInputs["key-id"] = id
	signatureInputs["signature-salt"] = string(crypto.Base64Encode(salt))
	container.Data.Options.SignatureInputs = signatureInputs
	// Force a clear of any existing signature values as that doesn't make sense
	container.Data.Options.Signature = ""

	containerJson := container.Dump()

	if err := crypto.HMAC([]byte(containerJson), newKey, signature); err != nil {
		return fmt.Errorf("Could not HMAC container: %s", err)
	}

	if signature.Message != containerJson {
		return fmt.Errorf("Authenticated message doesn't match")
	}

	container.Data.Options.Signature = signature.Signature
	return nil
}

func (entity *Entity) VerifyAuthentication(container *document.Container, key string) error {
	rawKey, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("Could not decode key: %s", err)
	}

	salt, err := crypto.Base64Decode([]byte(container.Data.Options.SignatureInputs["signature-salt"]))
	if err != nil {
		fmt.Errorf("Could not base64 decode signature salt: %s", err)
	}

	newKey, _, err := crypto.ExpandKey(rawKey, salt)
	if err != nil {
		return fmt.Errorf("Could not expand key: %s", err)
	}
	mac := crypto.NewSignature(crypto.SignatureModeSha256Hmac)

	mac.Signature = container.Data.Options.Signature
	container.Data.Options.Signature = ""

	if err := crypto.HMACVerify([]byte(container.Dump()), newKey, mac); err != nil {
		return fmt.Errorf("Couldn't verify container: %s", err)
	} else {
		return nil
	}
}

func (entity *Entity) Verify(container *document.Container) error {

	if container.IsSigned() == false {
		return fmt.Errorf("Container isn't signed")
	}

	signature := new(crypto.Signed)
	signature.Signature = container.Data.Options.Signature

	container.Data.Options.Signature = ""
	containerJson := container.Dump()
	signature.Message = containerJson

	if err := crypto.Verify(signature, entity.Data.Body.PublicSigningKey); err != nil {
		return fmt.Errorf("Could not verify org container signature: %s", err)
	} else {
		return nil
	}
}

func (entity *Entity) Decrypt(container *document.Container) (string, error) {
	if container.IsEncrypted() == false {
		return "", fmt.Errorf("Container isn't encrypted")
	}

	id := entity.Data.Body.Id
	key := entity.Data.Body.PrivateEncryptionKey
	if decryptedJson, err := container.Decrypt(id, key); err != nil {
		return "", fmt.Errorf("Could not decrypt: %s", err)
	} else {
		return decryptedJson, nil
	}
}

func (entity *Entity) Public() (*Entity, error) {
	selfJson := entity.Dump()
	publicEntity, err := New(selfJson)
	if err != nil {
		return nil, fmt.Errorf("Could not create public entity: %s", err)
	}
	publicEntity.Data.Body.PrivateSigningKey = ""
	publicEntity.Data.Body.PrivateEncryptionKey = ""
	return publicEntity, nil
}

func (entity *Entity) SignString(content string) (*document.Container, error) {
	container, err := document.NewContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("Could not create container: %s", err)
	}
	container.Data.Options.Source = entity.Data.Body.Id
	container.Data.Body = content
	if err := entity.Sign(container); err != nil {
		return nil, fmt.Errorf("Could not sign container: %s", err)
	} else {
		return container, nil
	}
}

func (entity *Entity) AuthenticateString(content, id, key string) (*document.Container, error) {
	container, err := document.NewContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("Could not create container: %s", err)
	}
	container.Data.Options.Source = entity.Data.Body.Id
	container.Data.Body = content
	if err := entity.Authenticate(container, id, key); err != nil {
		return nil, fmt.Errorf("Could not sign container: %s", err)
	} else {
		return container, nil
	}
}

func (entity *Entity) Encrypt(content string, entities interface{}) (*document.Container, error) {
	encryptionKeys := make(map[string]string)

	switch t := entities.(type) {
	case []*Entity:
		for _, e := range entities.([]*Entity) {
			encryptionKeys[e.Data.Body.Id] = e.Data.Body.PublicEncryptionKey
		}
	case nil:
		encryptionKeys[entity.Data.Body.Id] = entity.Data.Body.PublicEncryptionKey
	default:
		return nil, fmt.Errorf("Invalid entities given: %T", t)
	}

	container, err := document.NewContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("Could not create container: %s", err)
	}

	container.Data.Options.Source = entity.Data.Body.Id
	if err := container.Encrypt(content, encryptionKeys); err != nil {
		return nil, fmt.Errorf("Could not encrypt container: %s", err)
	}
	return container, nil
}

func (entity *Entity) EncryptThenSignString(content string, entities interface{}) (*document.Container, error) {

	container, err := entity.Encrypt(content, entities)
	if err != nil {
		return nil, fmt.Errorf("Couldn't encrypt content: %s", err)
	}

	if err := entity.Sign(container); err != nil {
		return nil, fmt.Errorf("Could not sign container: %s", err)
	}

	return container, nil
}

func (entity *Entity) EncryptThenAuthenticateString(content string, entities interface{}, id, key string) (*document.Container, error) {
	container, err := entity.Encrypt(content, entities)
	if err != nil {
		return nil, fmt.Errorf("Couldn't encrypt content: %s", err)
	}
	if err := entity.Authenticate(container, id, key); err != nil {
		return nil, fmt.Errorf("Could not authenticate container: %s", err)
	}
	return container, nil
}

func (entity *Entity) VerifyThenDecrypt(container *document.Container) (string, error) {
	if err := entity.Verify(container); err != nil {
		return "", fmt.Errorf("Could not verify container: %s", err)
	}

	content, err := entity.Decrypt(container)
	if err != nil {
		return "", fmt.Errorf("Could not decrypt container: %s", err)
	}
	return content, nil

}

func (entity *Entity) VerifyAuthenticationThenDecrypt(container *document.Container, key string) (string, error) {
	if err := entity.VerifyAuthentication(container, key); err != nil {
		return "", fmt.Errorf("Could not verify container: %s", err)
	}

	content, err := entity.Decrypt(container)
	if err != nil {
		return "", fmt.Errorf("Could not decrypt container: %s", err)
	}
	return content, nil
}
