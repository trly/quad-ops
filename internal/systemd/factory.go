package systemd

// DefaultFactory provides default implementations of all systemd interfaces.
type DefaultFactory struct {
	connectionFactory ConnectionFactory
	contextProvider   ContextProvider
	textCaser         TextCaser
	unitManager       UnitManager
	orchestrator      Orchestrator
}

// NewDefaultFactory creates a new default factory with all default implementations.
func NewDefaultFactory() *DefaultFactory {
	connectionFactory := NewDefaultConnectionFactory()
	contextProvider := NewDefaultContextProvider()
	textCaser := NewDefaultTextCaser()
	unitManager := NewDefaultUnitManager(connectionFactory, contextProvider, textCaser)
	orchestrator := NewDefaultOrchestrator(unitManager)

	return &DefaultFactory{
		connectionFactory: connectionFactory,
		contextProvider:   contextProvider,
		textCaser:         textCaser,
		unitManager:       unitManager,
		orchestrator:      orchestrator,
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

// Package-level convenience functions for backward compatibility

var defaultFactory = NewDefaultFactory()

// GetDefaultUnitManager returns the default unit manager instance.
func GetDefaultUnitManager() UnitManager {
	return defaultFactory.GetUnitManager()
}

// GetDefaultOrchestrator returns the default orchestrator instance.
func GetDefaultOrchestrator() Orchestrator {
	return defaultFactory.GetOrchestrator()
}
