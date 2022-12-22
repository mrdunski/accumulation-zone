//go:generate mockgen -destination=mock_glacier/connection.go . Cli
package glacier

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	"io"
	"time"
)

type Cli interface {
	DeleteArchive(input *glacier.DeleteArchiveInput) (*glacier.DeleteArchiveOutput, error)
	UploadArchive(input *glacier.UploadArchiveInput) (*glacier.ArchiveCreationOutput, error)
	ListJobs(input *glacier.ListJobsInput) (*glacier.ListJobsOutput, error)
	InitiateJob(input *glacier.InitiateJobInput) (*glacier.InitiateJobOutput, error)
	DescribeJob(input *glacier.DescribeJobInput) (*glacier.JobDescription, error)
	GetJobOutput(input *glacier.GetJobOutputInput) (*glacier.GetJobOutputOutput, error)
}

type Connection struct {
	glacier   Cli
	accountId string
	vaultName string
}

func NewConnection(cli Cli, vaultName, accountId string) Connection {
	return Connection{
		glacier:   cli,
		vaultName: vaultName,
		accountId: accountId,
	}
}

func OpenConnection(cfg VaultConfig) (*Connection, error) {
	ses, err := session.NewSession(&aws.Config{
		Credentials: cfg.credentials(),
		Region:      &cfg.RegionId,
	})

	if err != nil {
		return nil, err
	}

	return &Connection{
		glacier:   glacier.New(ses),
		vaultName: cfg.VaultName,
		accountId: cfg.AccountId,
	}, nil
}

func (c *Connection) Process(committer model.ChangeCommitter, changes model.Changes) error {
	for _, change := range changes.Additions {
		id, err := c.processAdd(change)
		if err != nil {
			return err
		}
		err = committer.CommitAdd(id, change)
		if err != nil {
			return err
		}
	}

	for _, change := range changes.Deletions {
		id, err := c.processDelete(change)
		if err != nil {
			return err
		}
		err = committer.CommitDelete(id, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Connection) processAdd(change model.FileAdded) (string, error) {
	id, err := c.Upload(change)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (c *Connection) processDelete(change model.FileDeleted) (string, error) {
	if change.ChangeId() == "" {
		return "", nil
	}
	err := c.Delete(change.ChangeId())
	if err != nil {
		return "", err
	}

	return change.ChangeId(), nil
}

func (c *Connection) Delete(id string) error {
	input := &glacier.DeleteArchiveInput{
		ArchiveId: &id,
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
	}

	_, err := c.glacier.DeleteArchive(input)
	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) Upload(file model.FileWithContent) (string, error) {
	content, err := file.Content()
	if err != nil {
		return "", err
	}

	checksum := file.Hash()
	if checksum == "" {
		return "", nil
	}

	input := &glacier.UploadArchiveInput{
		ArchiveDescription: aws.String(file.Path()),
		Body:               aws.ReadSeekCloser(content),
		Checksum:           &checksum,
		AccountId:          &c.accountId,
		VaultName:          &c.vaultName,
	}

	arch, err := c.glacier.UploadArchive(input)
	if err != nil {
		return "", err
	}

	return *arch.ArchiveId, nil
}

func (c *Connection) getInventoryJobOutput() ([]byte, error) {
	job, err := c.FindNewestInventoryJob()
	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, errors.New("there are no running or completed inventory jobs to print")
	}

	for job.StatusCode == nil || *job.StatusCode == "InProgress" {
		fmt.Printf(`job [%s] "%s" has status %s
`, flatString(job.JobId), flatString(job.JobDescription), flatString(job.StatusCode))
		job, err = c.FindNewestInventoryJob()
		if err != nil {
			return nil, err
		}
		time.Sleep(1 * time.Second)
	}

	if *job.StatusCode == "Failed" {
		return nil, fmt.Errorf("inventory job failed: %s", flatString(job.StatusMessage))
	}

	return c.GetInventoryJobOutput(*job.JobId)
}

func (c *Connection) PrintInventory() error {
	output, err := c.getInventoryJobOutput()
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(output))
	return nil
}

func (c *Connection) FindNewestInventoryJob() (*glacier.JobDescription, error) {
	input := glacier.ListJobsInput{
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
	}

	jobs, err := c.glacier.ListJobs(&input)
	if err != nil {
		return nil, err
	}

	var newest *glacier.JobDescription
	var newestCreation time.Time

	for _, job := range jobs.JobList {
		creation := parseCreationDate(job.CreationDate)
		if job.InventoryRetrievalParameters != nil && creation.After(newestCreation) {
			newest = job
			newestCreation = creation
		}
	}

	return newest, nil
}

func parseCreationDate(date *string) time.Time {
	if date == nil {
		return time.Now()
	}

	parsed, err := time.Parse(time.RFC3339, *date)
	if err != nil {
		return time.Now()
	}

	return parsed
}

func (c *Connection) CreateInventoryJob() (*glacier.JobDescription, error) {
	input := glacier.InitiateJobInput{
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
		JobParameters: &glacier.JobParameters{
			Description: aws.String("Update inventory"),
			Format:      aws.String("JSON"),
			Type:        aws.String("inventory-retrieval"),
		},
	}

	output, err := c.glacier.InitiateJob(&input)
	if err != nil {
		return nil, err
	}

	describe := glacier.DescribeJobInput{
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
		JobId:     output.JobId,
	}

	job, err := c.glacier.DescribeJob(&describe)
	if err != nil {
		return nil, err
	}

	return job, err
}

func (c *Connection) GetInventoryJobOutput(jobId string) ([]byte, error) {
	input := glacier.GetJobOutputInput{
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
		JobId:     &jobId,
	}

	output, err := c.glacier.GetJobOutput(&input)
	if err != nil {
		return nil, err
	}

	return getBody(output)
}

func (c *Connection) getInventory() (inventory, error) {
	rawOutput, err := c.getInventoryJobOutput()

	if err != nil {
		return inventory{}, fmt.Errorf("failed to get newest inventory job: %w", err)
	}

	i, err := unmarshalInventory(rawOutput)

	if err != nil {
		return inventory{}, fmt.Errorf("unsupported inventory format: %w", err)
	}

	return i, nil
}

func (c *Connection) ListInventoryAllFiles() ([]model.IdentifiableHashedFile, error) {
	i, err := c.getInventory()
	if err != nil {
		return nil, err
	}

	return i.asIdentifiableHashedFiles(), nil
}

func (c *Connection) ListInventoryNewestFiles() (map[string]model.IdentifiableHashedFile, error) {
	i, err := c.getInventory()
	if err != nil {
		return nil, err
	}

	return i.newestHashFiles(), nil
}

func (c *Connection) AddInventoryToIndex(idx index.Index) error {
	files, err := c.ListInventoryAllFiles()
	if err != nil {
		return err
	}

	err = idx.Clear()
	if err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}
	for _, file := range files {
		if err := idx.CommitAdd(file.ChangeId(), file); err != nil {
			_ = idx.Clear()
			return fmt.Errorf("failed to add to index [%s]: %w", file.Path(), err)
		}
	}

	return nil
}

func getBody(output *glacier.GetJobOutputOutput) (_ []byte, err error) {
	body := output.Body
	defer func(body io.ReadCloser) {
		err = body.Close()
	}(body)
	result, err := io.ReadAll(body)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func flatString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
