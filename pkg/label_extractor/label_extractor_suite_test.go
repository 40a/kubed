package label_extractor

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLabelExtractor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LabelExtractor Suite")
}
