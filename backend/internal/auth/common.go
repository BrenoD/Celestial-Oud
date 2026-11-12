package auth

// commonPasswords is a short blocklist of the most breached passwords
// (NCSC "PwnedPasswordsTop100k" head), lower-cased, 8+ characters only.
var commonPasswords = func() map[string]bool {
	m := map[string]bool{}
	for _, p := range []string{
		"password", "password1", "password12", "password123", "password1234", "passw0rd", "p@ssw0rd", "p@ssword",
		"12345678", "123456789", "1234567890", "12345678910", "123123123", "11111111", "111111111", "00000000",
		"87654321", "88888888", "99999999", "12341234", "1q2w3e4r", "1q2w3e4r5t", "qwertyuiop", "qwerty123", "qwerty12",
		"qwertyui", "asdfghjkl", "zxcvbnm1", "iloveyou", "iloveyou1", "sunshine", "princess", "football", "baseball",
		"welcome1", "welcome123", "superman", "starwars", "trustno1", "whatever", "liverpool", "chelsea1", "arsenal1",
		"letmein1", "letmein123", "monkey123", "dragon123", "abc12345", "abcd1234", "aa123456", "admin123", "administrator",
		"changeme", "computer", "internet", "michelle", "jennifer", "corvette", "mercedes", "samsung1", "football1",
		"charlie1", "shadow123", "master123", "access14", "q1w2e3r4", "q1w2e3r4t5", "zaq12wsx", "1qaz2wsx", "qazwsxedc",
		"celestial", "celestialoud", "perfume1", "fragrance", "oudoudoud",
	} {
		m[p] = true
	}
	return m
}()
