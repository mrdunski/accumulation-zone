package glacier

import "github.com/aws/aws-sdk-go/aws/credentials"

type VaultConfig struct {
	AccountId string `env:"ACCOUNT_ID" help:"AWS account id" required:""`
	RegionId  string `env:"REGION_ID" help:"AWS region" required:"" placeholder:"sl-bottom-0"`
	VaultName string `env:"VAULT_NAME" help:"AWS Glacier VaultConfig" required:""`
	KeyId     string `env:"KEY_ID" help:"AWS Access Key Id" required:""`
	KeySecret string `env:"KEY_SECRET" help:"AWS Access Key Secret" required:""`
}

func (c VaultConfig) credentials() *credentials.Credentials {
	return credentials.NewStaticCredentials(c.KeyId, c.KeySecret, "")
}
