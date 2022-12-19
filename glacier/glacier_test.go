package glacier_test

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	awsGlacier "github.com/aws/aws-sdk-go/service/glacier"
	"github.com/golang/mock/gomock"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/glacier/mock_glacier"
	"github.com/mrdunski/accumulation-zone/model"
	"github.com/mrdunski/accumulation-zone/model/mock_model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"reflect"
	"strings"
	"testing"
)

const (
	testVaultName   = "testVault"
	testAccountId   = "testAccount"
	testFileContent = "testContent"
	testFileHash    = "mockedHash"
	testFilePath    = "mockedPath"
)

func TestGlacier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "glacier")
}

var _ = Describe("Connection", func() {
	var glacierCli *mock_glacier.MockCli
	var connection glacier.Connection

	BeforeEach(func() {
		glacierCli = mock_glacier.NewMockCli(gomock.NewController(GinkgoT()))
		connection = glacier.NewConnection(glacierCli, testVaultName, testAccountId)
	})

	Describe("Process", func() {
		var committer *mock_model.MockChangeCommitter
		var exampleFile *mock_glacier.MockFileWithContent

		BeforeEach(func() {
			committer = mock_model.NewMockChangeCommitter(gomock.NewController(GinkgoT()))
			exampleFile = mock_glacier.NewMockFileWithContent(gomock.NewController(GinkgoT()))
			exampleFile.EXPECT().Hash().AnyTimes().Return(testFileHash)
			exampleFile.EXPECT().Path().AnyTimes().Return(testFilePath)
			exampleFile.EXPECT().
				Content().
				AnyTimes().
				DoAndReturn(func() (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(testFileContent)), nil
				})
		})

		It("handles add", func() {
			change := model.Change{ChangeType: model.Added, HashedFile: exampleFile}
			committer.EXPECT().CommitChange("testArchive1", change).Return(nil)
			glacierCli.EXPECT().
				UploadArchive(
					ArchiveEq{
						VaultName:          aws.String(testVaultName),
						AccountId:          aws.String(testAccountId),
						Checksum:           aws.String(testFileHash),
						ArchiveDescription: aws.String(testFilePath),
						Body:               aws.ReadSeekCloser(strings.NewReader(testFileContent)),
					},
				).
				Return(&awsGlacier.ArchiveCreationOutput{
					ArchiveId: aws.String("testArchive1"),
				}, nil)

			err := connection.Process(committer, []model.Change{change})

			Expect(err).NotTo(HaveOccurred())
		})

		It("handles delete", func() {
			change := model.Change{ChangeType: model.Deleted, HashedFile: FileWithChangeId{
				changeId:   "deletedArchive1",
				HashedFile: exampleFile,
			}}
			committer.EXPECT().CommitChange("deletedArchive1", change).Return(nil)
			glacierCli.EXPECT().
				DeleteArchive(
					gomock.Eq(&awsGlacier.DeleteArchiveInput{
						ArchiveId: aws.String("deletedArchive1"),
						AccountId: aws.String(testAccountId),
						VaultName: aws.String(testVaultName),
					}),
				).
				Return(&awsGlacier.DeleteArchiveOutput{}, nil)

			err := connection.Process(committer, []model.Change{change})
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles commit err", func() {
			change := model.Change{ChangeType: model.Deleted, HashedFile: FileWithChangeId{
				changeId:   "deletedArchive1",
				HashedFile: exampleFile,
			}}
			commitErr := errors.New("something bad")
			committer.EXPECT().CommitChange(gomock.Any(), gomock.Any()).Return(commitErr)
			glacierCli.EXPECT().DeleteArchive(gomock.Any()).AnyTimes().Return(&awsGlacier.DeleteArchiveOutput{}, nil)

			err := connection.Process(committer, []model.Change{change})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, commitErr)).To(BeTrue())
		})

		It("handles invalid change event", func() {
			file := mock_model.NewMockHashedFile(gomock.NewController(GinkgoT()))
			change := model.Change{ChangeType: model.Deleted, HashedFile: file}

			err := connection.Process(committer, []model.Change{change})
			Expect(err).To(HaveOccurred())
		})

		It("handles file delete without id", func() {
			change := model.Change{ChangeType: model.Deleted, HashedFile: FileWithChangeId{
				changeId:   "",
				HashedFile: exampleFile,
			}}
			committer.EXPECT().CommitChange("", change).Return(nil)

			err := connection.Process(committer, []model.Change{change})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("FindNewestInventoryJob", func() {
		It("should return nil when there are no jobs", func() {
			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{}, nil)

			job, err := connection.FindNewestInventoryJob()

			Expect(job).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the only job", func() {
			aJob := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}

			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&aJob}}, nil)

			job, err := connection.FindNewestInventoryJob()

			Expect(job).To(Equal(&aJob))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the newest job", func() {
			olderJob1 := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				CreationDate:                 aws.String("2015-12-20T00:00:08Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}
			newerJob := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				CreationDate:                 aws.String("2015-12-20T00:01:08Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}
			olderJob2 := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				CreationDate:                 aws.String("2015-12-20T00:01:07Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}

			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&olderJob1, &newerJob, &olderJob2}}, nil)

			job, err := connection.FindNewestInventoryJob()

			Expect(job).To(Equal(&newerJob))
			Expect(err).NotTo(HaveOccurred())
		})

		It("ignores wrong type of job", func() {
			correctJob := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				CreationDate:                 aws.String("2015-12-20T00:00:08Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}
			WrongJob := awsGlacier.JobDescription{
				JobId:        aws.String("testJob1"),
				CreationDate: aws.String("2015-12-20T00:01:08Z"),
			}

			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&correctJob, &WrongJob}}, nil)

			job, err := connection.FindNewestInventoryJob()

			Expect(job).To(Equal(&correctJob))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CreateInventoryJob", func() {
		It("creates new job", func() {
			glacierCli.EXPECT().InitiateJob(TestingInventoryRetrievalJob{}).Return(&awsGlacier.InitiateJobOutput{JobId: aws.String("testJob1")}, nil)
			glacierCli.EXPECT().DescribeJob(&awsGlacier.DescribeJobInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
				JobId:     aws.String("testJob1"),
			}).AnyTimes().Return(&awsGlacier.JobDescription{JobId: aws.String("testJob1")}, nil)
			job, err := connection.CreateInventoryJob()

			Expect(job.JobId).To(Equal(aws.String("testJob1")))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("PrintInventory", func() {
		It("prints inventory when job has finished", func() {
			aJob := awsGlacier.JobDescription{
				JobId:                        aws.String("aJob"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
				StatusCode:                   aws.String("Succeeded"),
			}
			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&aJob}}, nil)
			glacierCli.EXPECT().GetJobOutput(gomock.Eq(&awsGlacier.GetJobOutputInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
				JobId:     aws.String("aJob"),
			})).Return(&awsGlacier.GetJobOutputOutput{Body: io.NopCloser(strings.NewReader("{}"))}, nil)

			err := connection.PrintInventory()

			Expect(err).NotTo(HaveOccurred())
		})

		It("handles failed job", func() {
			aJob := awsGlacier.JobDescription{
				JobId:                        aws.String("aJob"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
				StatusCode:                   aws.String("Failed"),
				StatusMessage:                aws.String("bad weather"),
			}
			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&aJob}}, nil)

			err := connection.PrintInventory()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errors.New("inventory job failed: bad weather")))
		})
	})
})

type ArchiveEq awsGlacier.UploadArchiveInput

func (m ArchiveEq) Matches(x interface{}) bool {
	input, ok := x.(*awsGlacier.UploadArchiveInput)
	if !ok {
		return false
	}

	mBody, err := io.ReadAll(m.Body)
	if err != nil {
		return false
	}

	inBody, err := io.ReadAll(input.Body)
	if err != nil {
		return false
	}

	return *input.ArchiveDescription == *m.ArchiveDescription &&
		*input.Checksum == *m.Checksum &&
		*input.AccountId == *m.AccountId &&
		*input.VaultName == *m.VaultName &&
		reflect.DeepEqual(inBody, mBody)
}

func (m ArchiveEq) String() string {
	return awsutil.Prettify(m)
}

type TestingInventoryRetrievalJob struct{}

func (c TestingInventoryRetrievalJob) Matches(x interface{}) bool {
	input, ok := x.(*awsGlacier.InitiateJobInput)
	if !ok {
		return false
	}

	return testAccountId == *input.AccountId &&
		testVaultName == *input.VaultName &&
		"JSON" == *input.JobParameters.Format &&
		"inventory-retrieval" == *input.JobParameters.Type
}

func (c TestingInventoryRetrievalJob) String() string {
	return awsutil.Prettify(c)

}

type FileWithChangeId struct {
	model.HashedFile
	changeId string
}

func (f FileWithChangeId) ChangeId() string {
	return f.changeId
}
