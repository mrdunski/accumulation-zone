package glacier

import "github.com/aws/aws-sdk-go/aws/credentials"

type VaultConfig struct {
	AccountId string `env:"ACCOUNT_ID" help:"AWS account id" required:"" group:"AWS Glacier"`
	RegionId  string `env:"REGION_ID" help:"AWS region" required:"" placeholder:"sl-bottom-0" group:"AWS Glacier"`
	VaultName string `env:"VAULT_NAME" help:"AWS Glacier VaultConfig" required:"" group:"AWS Glacier"`
	KeyId     string `env:"KEY_ID" help:"AWS Access Key Id" required:"" group:"AWS IAM"`
	KeySecret string `env:"KEY_SECRET" help:"AWS Access Key Secret" required:"" group:"AWS IAM"`
}

func (c VaultConfig) credentials() *credentials.Credentials {
	return credentials.NewStaticCredentials(c.KeyId, c.KeySecret, "")
}

type RetrievalTier string

const (
	TierExpedited RetrievalTier = "Expedited"
	TierStandard  RetrievalTier = "Standard"
	TierBulk      RetrievalTier = "Bulk"
)

type ArchiveRetrievalOptions struct {
	Tier RetrievalTier `_:"
	                   " env:"RETRIEVAL_TIER" _:"
	                   " default:"Standard" _:"
	                   " enum:"Expedited,Standard,Bulk" _:"
	                   " help:"Class of retrieval
	* Expedited - quick and expensive - retrieval job finishes up to 5 minutes
	* Standard - moderate cost and retrieval times - up to 5 hours
	* Bulk - cost effective - up to 12 hours

For more info see https://docs.aws.amazon.com/amazonglacier/latest/dev/api-initiate-job-post.html"`
}
