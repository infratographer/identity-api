package storage

import (
	"io"
	"os"

	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/gidx"
	"gopkg.in/yaml.v3"
)

// SeedIssuer represents the seed data for a single issuer.
type SeedIssuer struct {
	OwnerID       gidx.PrefixedID   `yaml:"ownerID"`
	ID            gidx.PrefixedID   `yaml:"id"`
	Name          string            `yaml:"name"`
	URI           string            `yaml:"uri"`
	JWKSURI       string            `yaml:"jwksURI"`
	ClaimMappings map[string]string `yaml:"claimMappings"`
}

// SeedData represents the seed data for an identity-api instance on startup.
type SeedData struct {
	Issuers []SeedIssuer
}

func parseSeedData(path string) (SeedData, error) {
	f, err := os.Open(path)
	if err != nil {
		return SeedData{}, err
	}

	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return SeedData{}, err
	}

	var out SeedData

	err = yaml.Unmarshal(bytes, &out)
	if err != nil {
		return SeedData{}, err
	}

	return out, nil
}

// SeedDatabase seeds the database using the data at the given path
func SeedDatabase(config crdbx.Config, path string) error {
	data, err := parseSeedData(path)
	if err != nil {
		return err
	}

	_, err = newCRDBEngine(config, WithSeedData(data))

	return err
}
