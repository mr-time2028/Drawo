package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"drawo/config"
)

func TestNewContainer_Failures(t *testing.T) {
	cfg := config.Get()
	
    // Fail Database
	cfg.Database.Driver = "unknown"
	_, err := NewContainer(cfg)
    assert.Error(t, err)

    // Fail Cache
    cfg.Database.Driver = "postgres"
    cfg.Cache.Driver = "unknown"
    _, err = NewContainer(cfg)
    assert.Error(t, err)
}

func TestContainer_Health(t *testing.T) {
    c := &Container{}
    h := c.Health()
    assert.NotNil(t, h)
    assert.Empty(t, h)
}
