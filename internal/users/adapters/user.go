package adapters

type User struct {
	ID       string `firestore:"id"`
	Name     string `firestore:"name,omitempty"`
	Email    string `firestore:"email"`
	Password string `firestore:"password"`
}
