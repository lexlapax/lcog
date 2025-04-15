package entity

// EntityID represents a unique identifier for an entity in the system.
// Each entity has its own isolated memory space.
type EntityID string

// AccessLevel defines the visibility of memory records.
type AccessLevel int

const (
	// PrivateToUser indicates memory only accessible to a specific user within an entity
	PrivateToUser AccessLevel = iota
	
	// SharedWithinEntity indicates memory accessible to all users within the same entity
	SharedWithinEntity
)

// Context holds information about the current entity and user context.
type Context struct {
	// EntityID is mandatory and determines the memory isolation boundary
	EntityID EntityID
	
	// UserID is optional and used for PrivateToUser access level filtering
	UserID string
}

// NewContext creates a new Context with the specified entity ID and optional user ID.
func NewContext(entityID EntityID, userID string) Context {
	return Context{
		EntityID: entityID,
		UserID:   userID,
	}
}
