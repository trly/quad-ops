package systemd

import (
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
)

// DefaultFactory provides default implementations of all systemd interfaces.
type DefaultFactory struct {
	connectionFactory ConnectionFactory
	contextProvider   ContextProvider
	textCaser         TextCaser
	unitManager       UnitManager
	orchestrator      Orchestrator
	configProvider    config.Provider
	logger            log.Logger
}

// NewDefaultFactory creates a new default factory with all default implementations.
func NewDefaultFactory(configProvider config.Provider, logger log.Logger) *DefaultFactory {
	connectionFactory := NewConnectionFactory(logger)
	contextProvider := NewDefaultContextProvider()
	textCaser := NewDefaultTextCaser()
	runner := execx.NewRealRunner()
	unitManager := NewDefaultUnitManager(connectionFactory, contextProvider, textCaser, configProvider, logger, runner)
	orchestrator := NewDefaultOrchestrator(unitManager, connectionFactory, configProvider, logger)

	return &DefaultFactory{
		connectionFactory: connectionFactory,
		contextProvider:   contextProvider,
		textCaser:         textCaser,
		unitManager:       unitManager,
		orchestrator:      orchestrator,
		configProvider:    configProvider,
		logger:            logger,
	}
}

// GetConnectionFactory returns the connection factory.
func (f *DefaultFactory) GetConnectionFactory() ConnectionFactory {
	return f.connectionFactory
}

// GetContextProvider returns the context provider.
func (f *DefaultFactory) GetContextProvider() ContextProvider {
	return f.contextProvider
}

// GetTextCaser returns the text caser.
func (f *DefaultFactory) GetTextCaser() TextCaser {
	return f.textCaser
}

// GetUnitManager returns the unit manager.
func (f *DefaultFactory) GetUnitManager() UnitManager {
	return f.unitManager
}

// GetOrchestrator returns the orchestrator.
func (f *DefaultFactory) GetOrchestrator() Orchestrator {
	return f.orchestrator
}
