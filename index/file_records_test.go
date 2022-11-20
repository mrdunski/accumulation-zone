package index

import (
	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"os"
)

type entryMatcher struct {
	Entry
}

func (c entryMatcher) Match(actual interface{}) (success bool, err error) {
	entries, ok := actual.(entries)
	return ok && entries.hasEntryMatching(c.path, func(e Entry) bool {
		return e.hash == c.Hash() && e.changeId == c.changeId && e.recordDate.Equal(c.recordDate)
	}), nil
}

func (c entryMatcher) FailureMessage(_ interface{}) string {
	return ""
}

func (c entryMatcher) NegatedFailureMessage(_ interface{}) string {
	return ""
}

func haveEntry(entry Entry) types.GomegaMatcher {
	return entryMatcher{Entry: entry}
}

var _ = g.Describe("fileRecords", func() {
	var testFile *os.File

	g.BeforeEach(func() {
		temp, err := os.CreateTemp("", "*-testindex.log")
		Expect(err).NotTo(HaveOccurred())
		testFile = temp
	})

	g.AfterEach(func() {
		err := testFile.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	g.It("saves and loads single record", func() {
		records := fileRecords{
			filePath: testFile.Name(),
		}
		entry := NewEntry("test", "h123", "ch123")
		err := records.add(entry)
		Expect(err).NotTo(HaveOccurred())

		entries, err := records.loadEntries()
		Expect(err).NotTo(HaveOccurred())
		Expect(entries.flatten()).To(HaveLen(1))
		Expect(entries).To(haveEntry(entry))
	})

	g.It("saves and removes a record", func() {
		records := fileRecords{
			filePath: testFile.Name(),
		}
		entry := NewEntry("test", "h123", "ch123")
		err := records.add(entry)
		Expect(err).NotTo(HaveOccurred())

		err = records.remove(entry)
		Expect(err).NotTo(HaveOccurred())

		entries, err := records.loadEntries()
		Expect(err).NotTo(HaveOccurred())
		Expect(entries.flatten()).To(BeEmpty())
	})
})
