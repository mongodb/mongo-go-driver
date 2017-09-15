package auth

// Cred is a user's credential.
type Cred struct {
	Source      string
	Username    string
	Password    string
	PasswordSet bool
	Props       map[string]string
}
