package v1_test

import (
	"strings"

	v1 "github.com/9seconds/shebang/internal/config/v1"
	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite
}

func (suite *BaseTestSuite) Parse(doc string) (*v1.Config, error) {
	return v1.Parse(strings.NewReader(doc))
}
