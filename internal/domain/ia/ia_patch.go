package ia

// SetRegistry configura o ToolRegistry global do pacote.
var globalRegistry *ToolRegistry

func SetRegistry(r *ToolRegistry) {
	globalRegistry = r
}

func GetRegistry() *ToolRegistry {
	return globalRegistry
}
