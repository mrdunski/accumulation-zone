package glacier_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	awsGlacier "github.com/aws/aws-sdk-go/service/glacier"
	"github.com/golang/mock/gomock"
	"github.com/mrdunski/accumulation-zone/files"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/glacier/mock_glacier"
	. "github.com/mrdunski/accumulation-zone/gomega"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	"github.com/mrdunski/accumulation-zone/model/mock_model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testVaultName   = "testVault"
	testAccountId   = "testAccount"
	testFileContent = "testContent"
	testFileHash    = "mockedHash"
	testFilePath    = "mockedPath"
	testFileId      = "mockedId"
)

func TestGlacier(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "glacier")
}

var _ = Describe("Connection", func() {
	var glacierCli *mock_glacier.MockCli
	var connection glacier.Connection
	mockJobs := func(jobs ...awsGlacier.JobDescription) {
		jobList := make([]*awsGlacier.JobDescription, len(jobs))
		for i := range jobs {
			jobList[i] = &jobs[i]
		}

		glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
			AccountId: aws.String(testAccountId),
			VaultName: aws.String(testVaultName),
		})).Return(&awsGlacier.ListJobsOutput{JobList: jobList}, nil)
	}
	mockSingleJob := func(aJob awsGlacier.JobDescription, out ...string) {
		mockJobs(aJob)

		if len(out) == 1 {
			glacierCli.EXPECT().
				GetJobOutput(gomock.Eq(&awsGlacier.GetJobOutputInput{
					AccountId: aws.String(testAccountId),
					VaultName: aws.String(testVaultName),
					JobId:     aJob.JobId,
				})).
				MinTimes(1).
				Return(&awsGlacier.GetJobOutputOutput{Body: io.NopCloser(strings.NewReader(out[0]))}, nil)
		}

		if len(out) > 1 {
			panic(fmt.Sprintf("unsupported number of outputs: %v", out))
		}
	}
	mockNoJobs := func() {
		mockJobs()
	}
	mockSuccessfulInventoryJob := func(out string) {
		mockSingleJob(awsGlacier.JobDescription{
			JobId:                        aws.String("aJob"),
			InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			StatusCode:                   aws.String("Succeeded"),
		}, out)
	}

	BeforeEach(func() {
		glacierCli = mock_glacier.NewMockCli(gomock.NewController(GinkgoT()))
		connection = glacier.NewConnection(glacierCli, testVaultName, testAccountId)
	})

	Describe("Process", func() {
		var committer *mock_model.MockChangeCommitter
		var exampleFile *mock_model.MockFileWithContent

		BeforeEach(func() {
			committer = mock_model.NewMockChangeCommitter(gomock.NewController(GinkgoT()))
			exampleFile = mock_model.NewMockFileWithContent(gomock.NewController(GinkgoT()))
			exampleFile.EXPECT().Hash().AnyTimes().Return(testFileHash)
			exampleFile.EXPECT().Path().AnyTimes().Return(testFilePath)
			exampleFile.EXPECT().
				Content().
				AnyTimes().
				DoAndReturn(func() (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(testFileContent)), nil
				})
			exampleFile.EXPECT().Size().AnyTimes().Return(int64(100), nil)
		})

		It("handles add", func() {
			change := model.FileAdded{FileWithContent: exampleFile}
			committer.EXPECT().CommitAdd("testArchive1", change).Return(nil)
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

			err := connection.Process(committer, model.Changes{Additions: []model.FileAdded{change}})

			Expect(err).NotTo(HaveOccurred())
		})

		It("handles delete", func() {
			change := model.FileDeleted{IdentifiableHashedFile: FileWithChangeId{
				changeId:   "deletedArchive1",
				HashedFile: exampleFile,
			}}
			committer.EXPECT().CommitDelete("deletedArchive1", change).Return(nil)
			glacierCli.EXPECT().
				DeleteArchive(
					gomock.Eq(&awsGlacier.DeleteArchiveInput{
						ArchiveId: aws.String("deletedArchive1"),
						AccountId: aws.String(testAccountId),
						VaultName: aws.String(testVaultName),
					}),
				).
				Return(&awsGlacier.DeleteArchiveOutput{}, nil)

			err := connection.Process(committer, model.Changes{Deletions: []model.FileDeleted{change}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles commit err", func() {
			change := model.FileDeleted{IdentifiableHashedFile: FileWithChangeId{
				changeId:   "deletedArchive1",
				HashedFile: exampleFile,
			}}
			commitErr := errors.New("something bad")
			committer.EXPECT().CommitDelete(gomock.Any(), gomock.Any()).Return(commitErr)
			glacierCli.EXPECT().DeleteArchive(gomock.Any()).AnyTimes().Return(&awsGlacier.DeleteArchiveOutput{}, nil)

			err := connection.Process(committer, model.Changes{Deletions: []model.FileDeleted{change}})
			Expect(err).To(WrapError(commitErr))
		})
	})

	Describe("FindNewestInventoryJob", func() {
		It("should return nil when there are no jobs", func() {
			mockNoJobs()

			job, err := connection.FindNewestInventoryJob()

			Expect(job).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the only job", func() {
			aJob := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob1"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}
			mockSingleJob(aJob)

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
				JobId:                        aws.String("testJob2"),
				CreationDate:                 aws.String("2015-12-20T00:01:08Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}
			olderJob2 := awsGlacier.JobDescription{
				JobId:                        aws.String("testJob3"),
				CreationDate:                 aws.String("2015-12-20T00:01:07Z"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
			}

			mockJobs(olderJob1, newerJob, olderJob2)

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
			wrongJob := awsGlacier.JobDescription{
				JobId:        aws.String("testJob1"),
				CreationDate: aws.String("2015-12-20T00:01:08Z"),
			}

			mockJobs(correctJob, wrongJob)

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

	Describe("InventoryContent", func() {
		It("prints inventory when job has finished", func() {
			mockSuccessfulInventoryJob("{}")

			out, err := connection.InventoryContent()

			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("{}"))
		})

		It("handles failed job", func() {
			mockSingleJob(awsGlacier.JobDescription{
				JobId:                        aws.String("aJob"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
				StatusCode:                   aws.String("Failed"),
				StatusMessage:                aws.String("bad weather"),
			})

			out, err := connection.InventoryContent()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("job failed: bad weather"))
			Expect(out).To(BeEmpty())
		})
	})

	Describe("ListInventoryAllFiles", func() {
		It("returns error when there is no inventory job", func() {
			mockNoJobs()

			files, err := connection.ListInventoryAllFiles()

			Expect(err).To(HaveOccurred())
			Expect(files).To(BeEmpty())
		})

		It("returns empty list for empty inventory", func() {
			mockSuccessfulInventoryJob("{}")

			files, err := connection.ListInventoryAllFiles()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(BeEmpty())
		})

		It("handles job in progress", func() {
			inProgress := awsGlacier.JobDescription{
				JobId:                        aws.String("aJob"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
				StatusCode:                   aws.String("InProgress"),
			}
			done := awsGlacier.JobDescription{
				JobId:                        aws.String("aJob"),
				InventoryRetrievalParameters: &awsGlacier.InventoryRetrievalJobDescription{},
				StatusCode:                   aws.String("Succeeded"),
			}

			firstCall := true

			glacierCli.EXPECT().ListJobs(gomock.Eq(&awsGlacier.ListJobsInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
			})).Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&inProgress}}, nil)
			glacierCli.EXPECT().DescribeJob(gomock.Eq(&awsGlacier.DescribeJobInput{
				AccountId: aws.String(testAccountId),
				VaultName: aws.String(testVaultName),
				JobId:     aws.String("aJob"),
			})).Times(2).DoAndReturn(func(_ interface{}) (*awsGlacier.JobDescription, error) {
				if firstCall {
					firstCall = false
					return &inProgress, nil
				}
				return &done, nil
			})

			glacierCli.EXPECT().
				GetJobOutput(gomock.Any()).
				Return(&awsGlacier.GetJobOutputOutput{Body: io.NopCloser(strings.NewReader("{}"))}, nil)

			files, err := connection.ListInventoryAllFiles()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(BeEmpty())
		})

		It("handles invalid output", func() {
			mockSuccessfulInventoryJob(`---
ArchiveList:
  - ArchiveId: test`)

			files, err := connection.ListInventoryAllFiles()

			syntaxErr := &json.SyntaxError{}
			Expect(errors.As(err, &syntaxErr)).To(BeTrue())
			Expect(files).To(BeEmpty())
		})

		DescribeTable("handles an archive", func(jobOut string, expectedPath, expectedHash, expectedId string) {
			mockSuccessfulInventoryJob(jobOut)

			files, err := connection.ListInventoryAllFiles()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ConsistOf(MatchingFile{Path: expectedPath, Hash: expectedHash, Id: expectedId}))
		},
			Entry("with id", `{"ArchiveList":[{"ArchiveId":"testId"}]}`, "", "", "testId"),
			Entry("with path", `{"ArchiveList":[{"ArchiveDescription":"testPath"}]}`, "testPath", "", ""),
			Entry("with hash", `{"ArchiveList":[{"SHA256TreeHash":"testHash"}]}`, "", "testHash", ""),
		)
	})

	Context("with fixture of glacier inventory", func() {

		BeforeEach(func() {
			_, f, _, _ := runtime.Caller(0)
			dirPath := filepath.Dir(f)
			fixturePath := filepath.Join(dirPath, "mock_glacier", "inventory-fixture.json")
			content, err := os.ReadFile(fixturePath)
			if err != nil {
				GinkgoT().Fatalf("can't read fixture %v", err)
			}
			mockSuccessfulInventoryJob(string(content))
		})

		It("list all inventory files", func() {
			files, err := connection.ListInventoryAllFiles()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(35))
			Expect(files).To(ContainElement(MatchingFile{Path: "Screenshot_1586941714.png", Hash: "245a22d183de84e2844712706570d8d26ff1c4a089bd122534bdc2c9159e7587", Id: "Ggg-H3s-U09N49_dPuFmh80xLYI6kaoqtqPfY9KuYebiW0FVN6OIQBFNBpsh2iR8RonVEdR8hV1Uzuw0mLUTN5SiOG-F8gIdtF3KjnnHUqajp2KOlT0B7XERr501T6KN-bN6dsETgg"}))
			Expect(files).To(ContainElement(MatchingFile{Path: "Screenshot_1586941714.png", Hash: "245a22d183de84e2844712706570d8d26ff1c4a089bd122534bdc2c9159e7587", Id: "JOyWdcP28gbjIa1I7auPbn6nTjC7XiHVmX8R0mzvxmL0THzdrr_LGFRDfyqC6jVoRn6x9AgGYAWZIt5DAch6tIVYvLnqZkzlyVQ2PguieuFkUGJlAzWf8PYz3mou_SAvO9oRgGgHtQ"}))
		})

		It("list newest inventory files", func() {
			files, err := connection.ListInventoryNewestFiles()

			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(HaveLen(8))
			Expect(files).To(ContainElement(MatchingFile{Path: "dir/testfile1", Hash: "26637da1bd793f9011a3d304372a9ec44e36cc677d2bbfba32a2f31f912358fe", Id: "0rr2-lnYot72W0eJXFLbjL2U-E6Vx3KwL8NfrnvR6PO2J7lPfGYpyX95bfSTSOFBY47lwyssFyUBONyQAV5vM0mdvI76C54Mr6wB1swPyrTwHN4cdwwAAN_LKvf82hJbDC5qOSL2OA"}))
		})

		Describe("interacting with index", func() {
			var testDir string
			var idxFile *os.File
			BeforeEach(func() {
				temp, err := os.MkdirTemp(os.TempDir(), "idx-test-*")
				if err != nil {
					GinkgoT().Fatalf("can't create idx file: %v", err)
				}
				testDir = temp
				idxFile, err = os.CreateTemp(testDir, "idx-*.log")
				if err != nil {
					GinkgoT().Fatalf("can't create idx file: %v", err)
				}
			})

			AfterEach(func() {
				err := idxFile.Close()
				if err != nil {
					Fail("can't close idx file")
				}
			})

			It("creates working index", func() {
				idx, err := index.LoadIndexFile(idxFile.Name())
				err = connection.AddInventoryToIndex(idx)
				Expect(err).NotTo(HaveOccurred())

				tree, err := files.NewVolume(testDir).LoadTree()
				Expect(err).ToNot(HaveOccurred())

				changes := idx.CalculateChanges(tree)
				Expect(changes.Additions).ToNot(BeEmpty())
				Expect(changes.Deletions).ToNot(BeEmpty())
			})

			It("creates loadable index", func() {
				idx, err := index.LoadIndexFile(idxFile.Name())
				err = connection.AddInventoryToIndex(idx)
				Expect(err).NotTo(HaveOccurred())

				tree, err := files.NewVolume(testDir).LoadTree()
				Expect(err).ToNot(HaveOccurred())

				idx, err = index.LoadIndexFile(idxFile.Name())
				Expect(err).ToNot(HaveOccurred())

				changes := idx.CalculateChanges(tree)
				Expect(changes.Additions).ToNot(BeEmpty())
				Expect(changes.Deletions).ToNot(BeEmpty())
			})
		})
	})

	Describe("CreateArchiveJob", func() {

		Context("with example file", func() {
			var file *mock_model.MockIdentifiableHashedFile
			var job *awsGlacier.JobDescription
			var retrievalOptions = glacier.ArchiveRetrievalOptions{Tier: glacier.TierBulk}

			BeforeEach(func() {
				file = NewTestingFile()

				job = &awsGlacier.JobDescription{
					JobId: aws.String("aJob"),
				}
			})

			It("creates archive retrieval job", func() {
				glacierCli.EXPECT().InitiateJob(gomock.Eq(&awsGlacier.InitiateJobInput{
					AccountId: aws.String(testAccountId),
					VaultName: aws.String(testVaultName),
					JobParameters: &awsGlacier.JobParameters{
						Tier:        aws.String(string(retrievalOptions.Tier)),
						ArchiveId:   aws.String(testFileId),
						Description: aws.String(testFilePath),
						Type:        aws.String("archive-retrieval"),
					},
				})).Return(&awsGlacier.InitiateJobOutput{JobId: job.JobId}, nil)
				glacierCli.EXPECT().DescribeJob(&awsGlacier.DescribeJobInput{
					AccountId: aws.String(testAccountId),
					VaultName: aws.String(testVaultName),
					JobId:     job.JobId,
				}).Return(job, nil)

				job, err := connection.CreateArchiveJob(file, retrievalOptions)

				Expect(err).NotTo(HaveOccurred())
				Expect(job).ToNot(BeNil())
			})

			It("handles job description error", func() {
				descrErr := errors.New("something bad")
				glacierCli.EXPECT().InitiateJob(gomock.Any()).Return(&awsGlacier.InitiateJobOutput{JobId: job.JobId}, nil)
				glacierCli.EXPECT().DescribeJob(gomock.Any()).Return(nil, descrErr)

				job, err := connection.CreateArchiveJob(file, retrievalOptions)

				Expect(err).To(WrapError(descrErr))
				Expect(job).To(BeNil())
			})

			It("handles job creation error", func() {
				initErr := errors.New("something bad")
				glacierCli.EXPECT().InitiateJob(gomock.Any()).Return(nil, initErr)

				job, err := connection.CreateArchiveJob(file, retrievalOptions)

				Expect(err).To(WrapError(initErr))
				Expect(job).To(BeNil())
			})
		})

	})

	Describe("FindOrCreateArchiveJob", func() {
		var testingFile, otherFile *mock_model.MockIdentifiableHashedFile

		BeforeEach(func() {
			testingFile = NewTestingFile()
			otherFile = NewFile("otherId")
		})

		Context("with job for test file", func() {
			var existingJob, createdJob awsGlacier.JobDescription

			BeforeEach(func() {
				existingJob = awsGlacier.JobDescription{
					JobId:     aws.String("aJob"),
					ArchiveId: aws.String(testFileId),
				}
				createdJob = awsGlacier.JobDescription{
					JobId:     aws.String("someOtherJob"),
					ArchiveId: aws.String(testFileId),
				}

				glacierCli.EXPECT().ListJobs(gomock.Any()).AnyTimes().Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&existingJob}}, nil)
				glacierCli.EXPECT().DescribeJob(gomock.Eq(&awsGlacier.DescribeJobInput{AccountId: aws.String(testAccountId), VaultName: aws.String(testVaultName), JobId: existingJob.JobId})).AnyTimes().Return(&existingJob, nil)
				glacierCli.EXPECT().InitiateJob(gomock.Any()).AnyTimes().Return(&awsGlacier.InitiateJobOutput{JobId: aws.String("someOtherJob")}, nil)
				glacierCli.EXPECT().DescribeJob(gomock.Eq(&awsGlacier.DescribeJobInput{AccountId: aws.String(testAccountId), VaultName: aws.String(testVaultName), JobId: createdJob.JobId})).AnyTimes().Return(&createdJob, nil)
			})

			It("returns a existingJob for test file", func() {
				result, err := connection.FindOrCreateArchiveJob(testingFile, glacier.ArchiveRetrievalOptions{Tier: glacier.TierExpedited})

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(&existingJob))
			})

			It("creates a existingJob for other file", func() {
				result, err := connection.FindOrCreateArchiveJob(otherFile, glacier.ArchiveRetrievalOptions{Tier: glacier.TierStandard})

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(&createdJob))
			})
		})

		Context("with failing call for jobs", func() {
			connErr := errors.New("omg")

			BeforeEach(func() {
				glacierCli.EXPECT().ListJobs(gomock.Any()).AnyTimes().Return(nil, connErr)
			})

			It("handles err", func() {
				job, err := connection.FindOrCreateArchiveJob(testingFile, glacier.ArchiveRetrievalOptions{})

				Expect(err).To(WrapError(connErr))
				Expect(job).To(BeNil())
			})
		})
	})

	Describe("LoadContentFromGlacier", func() {
		Context("with job for test file", func() {
			var existingJob awsGlacier.JobDescription

			BeforeEach(func() {
				existingJob = awsGlacier.JobDescription{
					JobId:      aws.String("aJob"),
					ArchiveId:  aws.String(testFileId),
					StatusCode: aws.String("Succeeded"),
				}

				glacierCli.EXPECT().ListJobs(gomock.Any()).AnyTimes().Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&existingJob}}, nil)
				glacierCli.EXPECT().GetJobOutput(gomock.Eq(&awsGlacier.GetJobOutputInput{AccountId: aws.String(testAccountId), VaultName: aws.String(testVaultName), JobId: existingJob.JobId})).AnyTimes().Return(&awsGlacier.GetJobOutputOutput{Body: io.NopCloser(strings.NewReader(testFileContent))}, nil)
			})

			It("loads file content from job output", func() {
				file := NewTestingFile()
				fileFromGlacier, err := connection.LoadContentFromGlacier(file)

				Expect(err).ToNot(HaveOccurred())
				if Expect(fileFromGlacier).ToNot(BeNil()) {
					content, err := fileFromGlacier.Content()
					if Expect(err).NotTo(HaveOccurred()) {
						bytes, err := io.ReadAll(content)
						Expect(err).NotTo(HaveOccurred())
						Expect(bytes).To(Equal([]byte(testFileContent)))
					}
				}
			})

			It("handles job output error", func() {
				missingFile := NewFile("missing")
				fileFromGlacier, err := connection.LoadContentFromGlacier(missingFile)
				Expect(err).To(HaveOccurred())
				Expect(fileFromGlacier).To(BeNil())
			})
		})

		var exampleErr = errors.New("omg")
		DescribeTable("handle error", func(listErr, describeErr, outputErr, expectedFileErr, expectedFileContentErr error) {
			existingJob := awsGlacier.JobDescription{
				JobId:      aws.String("aJob"),
				ArchiveId:  aws.String(testFileId),
				StatusCode: aws.String("InProgress"),
			}
			finishedJob := awsGlacier.JobDescription{
				JobId:      aws.String("aJob"),
				ArchiveId:  aws.String(testFileId),
				StatusCode: aws.String("Succeeded"),
			}

			glacierCli.EXPECT().ListJobs(gomock.Any()).AnyTimes().Return(&awsGlacier.ListJobsOutput{JobList: []*awsGlacier.JobDescription{&existingJob}}, listErr)
			glacierCli.EXPECT().DescribeJob(gomock.Eq(&awsGlacier.DescribeJobInput{AccountId: aws.String(testAccountId), VaultName: aws.String(testVaultName), JobId: existingJob.JobId})).AnyTimes().Return(&finishedJob, describeErr)
			glacierCli.EXPECT().GetJobOutput(gomock.Eq(&awsGlacier.GetJobOutputInput{AccountId: aws.String(testAccountId), VaultName: aws.String(testVaultName), JobId: finishedJob.JobId})).AnyTimes().Return(&awsGlacier.GetJobOutputOutput{Body: io.NopCloser(strings.NewReader(testFileContent))}, outputErr)

			file := NewTestingFile()
			fileFromGlacier, err := connection.LoadContentFromGlacier(file)

			if expectedFileErr != nil {
				Expect(err).To(WrapError(expectedFileErr))
			} else if Expect(err).NotTo(HaveOccurred()) && Expect(fileFromGlacier).ToNot(BeNil()) {
				_, err := fileFromGlacier.Content()
				if expectedFileContentErr != nil {
					Expect(err).To(WrapError(expectedFileContentErr))
				}
			}
		},
			Entry("in ListJobs", exampleErr, nil, nil, exampleErr, nil),
			Entry("in DescribeJob", nil, exampleErr, nil, exampleErr, nil),
			Entry("in GetJobOutput", nil, nil, exampleErr, nil, exampleErr),
		)
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

type MatchingFile struct {
	Path string
	Hash string
	Id   string
}

func (expected MatchingFile) Match(actual interface{}) (bool, error) {
	if hashFile, ok := actual.(model.IdentifiableHashedFile); ok {
		return hashFile.Hash() == expected.Hash && hashFile.Path() == expected.Path && hashFile.ChangeId() == expected.Id, nil
	}

	return false, fmt.Errorf("unexpected type: %v", actual)
}

func (expected MatchingFile) FailureMessage(actual interface{}) string {
	hashFile := actual.(model.IdentifiableHashedFile)
	return fmt.Sprintf("expected: {%s, %s: %s} got {%s, %s: %s}", expected.Id, expected.Path, expected.Hash, hashFile.ChangeId(), hashFile.Path(), hashFile.Hash())
}

func (expected MatchingFile) NegatedFailureMessage(_ interface{}) string {
	return fmt.Sprintf("expected to not have: {%s, %s: %s}", expected.Id, expected.Path, expected.Hash)
}

func NewTestingFile() *mock_model.MockIdentifiableHashedFile {
	return NewFile(testFileId)
}

func NewFile(id string) *mock_model.MockIdentifiableHashedFile {
	file := mock_model.NewMockIdentifiableHashedFile(gomock.NewController(GinkgoT()))
	file.EXPECT().Path().AnyTimes().Return(testFilePath)
	file.EXPECT().Hash().AnyTimes().Return(testFileHash)
	file.EXPECT().ChangeId().AnyTimes().Return(id)

	return file
}
