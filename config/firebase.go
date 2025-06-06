package config

// ServiceAccount holds essential fields from your JSON key
type ServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

var FirebaseBucketName = "livewiremashariki-14998.appspot.com"
