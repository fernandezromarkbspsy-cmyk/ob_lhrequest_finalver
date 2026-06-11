package auth

type LoginPayload struct {
	LoginType string `json:"login_type" form:"login_type"`
	Email     string `json:"email" form:"email"`
	OpsID     string `json:"ops_id" form:"ops_id"`
	Password  string `json:"password" form:"password"`
}

type OTPPayload struct {
	Email string `json:"email" form:"email"`
	Code  string `json:"code" form:"code"`
}

type ChangePasswordPayload struct {
	CurrentPassword string `json:"current_password" form:"current_password"`
	NewPassword     string `json:"new_password" form:"new_password"`
}

type AppError struct {
	Code    int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}
