package users

type Payload struct {
	Name      string `json:"name" form:"name"`
	Role      string `json:"role" form:"role"`
	Email     string `json:"email" form:"email"`
	OpsID     string `json:"ops_id" form:"ops_id"`
	Password  string `json:"password" form:"password"`
	ActorRole string `json:"actor_role" form:"actor_role"`
}

type ListFilter struct {
	Page    int
	PerPage int
}

type AppError struct {
	Code    int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}
