//go:generate mockgen -destination=mock_glacier/connection.go . Cli,FileWithContent,ChangeIdHolder
package glacier

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/glacier"
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

type FileWithContent interface {
	Content() (io.ReadCloser, error)
	Hash() string
	Path() string
}

type ChangeIdHolder interface {
	ChangeId() string
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

func (c *Connection) Process(committer model.ChangeCommitter, changes []model.Change) error {
	for _, change := range changes {
		id, err := c.processChange(change)
		if err != nil {
			return err
		}
		err = committer.CommitChange(id, change)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Connection) processChange(change model.Change) (string, error) {

	switch change.ChangeType {
	case model.Added:
		fileWithContent, ok := change.HashedFile.(FileWithContent)
		if !ok {
			return "", errors.New("add change without content")
		}
		id, err := c.Upload(fileWithContent)
		if err != nil {
			return "", err
		}
		return id, nil
	case model.Deleted:
		changeIdHolder, ok := change.HashedFile.(ChangeIdHolder)
		if !ok {
			return "", errors.New("delete change without change id")
		}
		if changeIdHolder.ChangeId() == "" {
			return "", nil
		}
		err := c.Delete(changeIdHolder.ChangeId())
		if err != nil {
			return "", err
		}

		return changeIdHolder.ChangeId(), nil
	}

	return "", fmt.Errorf("unprocessable change: %v", change)
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

func (c *Connection) Upload(file FileWithContent) (string, error) {
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

func (c *Connection) PrintInventory() error {
	job, err := c.FindNewestInventoryJob()
	if err != nil {
		return err
	}

	if job == nil {
		return errors.New("there are no running or completed inventory jobs to print")
	}

	for job.StatusCode == nil || *job.StatusCode == "InProgress" {
		fmt.Printf(`job [%s] "%s" has status %s (%s)
`, flatString(job.JobId), flatString(job.JobDescription), flatString(job.StatusCode), flatString(job.StatusMessage))
		job, err = c.FindNewestInventoryJob()
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	if *job.StatusCode == "Failed" {
		return fmt.Errorf("inventory job failed: %s", flatString(job.StatusMessage))
	}

	output, err := c.GetInventoryJobOutput(*job.JobId)
	if err != nil {
		return err
	}
	fmt.Printf("%v", output)
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

func (c *Connection) GetInventoryJobOutput(jobId string) (string, error) {
	input := glacier.GetJobOutputInput{
		AccountId: &c.accountId,
		VaultName: &c.vaultName,
		JobId:     &jobId,
	}

	output, err := c.glacier.GetJobOutput(&input)
	if err != nil {
		return "", err
	}

	return getBody(output)
}

func getBody(output *glacier.GetJobOutputOutput) (_ string, err error) {
	body := output.Body
	defer func(body io.ReadCloser) {
		err = body.Close()
	}(body)
	result, err := io.ReadAll(body)

	if err != nil {
		return "", err
	}

	return string(result), nil
}

func flatString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
