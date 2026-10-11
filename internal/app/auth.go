package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type credentials struct {
	Username string `json:"username"`
	Hash     string `json:"hash"`
}
type attempt struct {
	Count int
	Until time.Time
}
type Security struct {
	store      *Store
	aead       cipher.AEAD
	setupToken string
	mu         sync.Mutex
	attempts   map[string]attempt
}

func NewSecurity(s *Store) (*Security, error) {
	p := filepath.Join(s.Dir, "secret.key")
	key, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		key = make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return nil, e
		}
		e = os.WriteFile(p, key, 0600)
	}
	if e != nil {
		return nil, e
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	a, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	sec := &Security{store: s, aead: a, attempts: map[string]attempt{}}
	if !sec.Initialized() {
		p := filepath.Join(s.Dir, "setup-token")
		t, e := os.ReadFile(p)
		if os.IsNotExist(e) {
			t = []byte(randomToken())
			e = os.WriteFile(p, t, 0600)
		}
		if e != nil {
			return nil, e
		}
		sec.setupToken = strings.TrimSpace(string(t))
	}
	return sec, nil
}
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func tokenHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (s *Security) Initialized() bool {
	var c credentials
	return s.store.Get("admin", &c) == nil && c.Hash != ""
}
func (s *Security) Seal(v string) string {
	if v == "" {
		return ""
	}
	n := make([]byte, s.aead.NonceSize())
	rand.Read(n)
	return "enc:" + base64.StdEncoding.EncodeToString(s.aead.Seal(n, n, []byte(v), []byte("arrscope-api-key-v1")))
}
func (s *Security) Open(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if !strings.HasPrefix(v, "enc:") {
		return "", fmt.Errorf("Ungültiger verschlüsselter API-Key")
	}
	b, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(v, "enc:"))
	if e != nil || len(b) < s.aead.NonceSize() {
		return "", fmt.Errorf("API-Key konnte nicht entschlüsselt werden")
	}
	n := s.aead.NonceSize()
	p, e := s.aead.Open(nil, b[:n], b[n:], []byte("arrscope-api-key-v1"))
	return string(p), e
}
func validPassword(p string) bool { return len(p) >= 12 && len(p) <= 72 }
func (s *Security) Setup(user, password, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Initialized() {
		return fmt.Errorf("Bereits eingerichtet")
	}
	if s.setupToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.setupToken)) != 1 {
		return fmt.Errorf("Einrichtungscode ist ungültig")
	}
	if len(strings.TrimSpace(user)) < 2 || len(user) > 64 || !validPassword(password) {
		return fmt.Errorf("Benutzername mindestens 2 Zeichen; Passwort 12–72 UTF-8-Bytes")
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return e
	}
	c := credentials{strings.TrimSpace(user), string(h)}
	b, _ := json.Marshal(c)
	if _, e = s.store.DB.Exec("INSERT INTO kv(key,value) VALUES('admin',?)", string(b)); e != nil {
		return e
	}
	os.Remove(filepath.Join(s.store.Dir, "setup-token"))
	s.setupToken = ""
	return nil
}
func (s *Security) Login(user, password, ip string) bool {
	return s.loginHash(user, password, ip) != ""
}

// The verified hash guards session creation against a concurrent password reset,
// including reset-password in another process sharing the database.
func (s *Security) loginHash(user, password, ip string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	a := s.attempts[ip]
	if now.Before(a.Until) && a.Count >= 5 {
		return ""
	}
	if now.After(a.Until) {
		a = attempt{Until: now.Add(15 * time.Minute)}
	}
	var c credentials
	s.store.Get("admin", &c)
	passwordValid := bcrypt.CompareHashAndPassword([]byte(c.Hash), []byte(password)) == nil
	valid := c.Username == user && passwordValid
	if !valid {
		a.Count++
		s.attempts[ip] = a
	} else {
		delete(s.attempts, ip)
	}
	if len(s.attempts) > 10000 {
		for k, v := range s.attempts {
			if now.After(v.Until) {
				delete(s.attempts, k)
			}
		}
	}
	if valid {
		return c.Hash
	}
	return ""
}
func remoteIP(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return r.RemoteAddr
	}
	return h
}
func (s *Security) Session(r *http.Request) (string, string, bool) {
	c, e := r.Cookie("arrscope_session")
	if e != nil {
		return "", "", false
	}
	hash := tokenHash(c.Value)
	var csrf string
	var expires int64
	e = s.store.DB.QueryRow("SELECT csrf,expires FROM sessions WHERE token=?", hash).Scan(&csrf, &expires)
	return hash, csrf, e == nil && time.Now().Unix() < expires
}

var errCredentialsChanged = errors.New("Anmeldedaten wurden geändert; erneut anmelden")

func (s *Security) LoginSession(w http.ResponseWriter, r *http.Request, user, password, ip string) (bool, error) {
	hash := s.loginHash(user, password, ip)
	if hash == "" {
		return false, nil
	}
	err := s.createSession(w, r, hash)
	if errors.Is(err, errCredentialsChanged) {
		return false, nil
	}
	return err == nil, err
}
func (s *Security) CreateSession(w http.ResponseWriter, r *http.Request) error {
	var c credentials
	if err := s.store.Get("admin", &c); err != nil {
		return err
	}
	return s.createSession(w, r, c.Hash)
}
func (s *Security) createSession(w http.ResponseWriter, r *http.Request, verifiedHash string) error {
	token, csrf := randomToken(), randomToken()
	s.store.DB.Exec("DELETE FROM sessions WHERE expires<?", time.Now().Unix())
	result, e := s.store.DB.Exec("INSERT INTO sessions(token,csrf,expires) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM kv WHERE key='admin' AND json_extract(value,'$.hash')=?)", tokenHash(token), csrf, time.Now().Add(24*time.Hour).Unix(), verifiedHash)
	if e != nil {
		return e
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return errCredentialsChanged
	}
	http.SetCookie(w, &http.Cookie{Name: "arrscope_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil || os.Getenv("ARRSCOPE_HTTPS") == "true", MaxAge: 86400})
	return nil
}
func (s *Security) Reset(password string) error { return s.reset(password, "") }
func (s *Security) reset(password, verifiedHash string) error {
	if !validPassword(password) {
		return fmt.Errorf("Passwort muss 12–72 UTF-8-Bytes lang sein")
	}
	var c credentials
	if e := s.store.Get("admin", &c); e != nil {
		return e
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return e
	}
	c.Hash = string(h)
	tx, e := s.store.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	b, e := json.Marshal(c)
	if e != nil {
		return e
	}
	result, e := tx.Exec("UPDATE kv SET value=? WHERE key='admin' AND (?='' OR json_extract(value,'$.hash')=?)", string(b), verifiedHash, verifiedHash)
	if e != nil {
		return e
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return errCredentialsChanged
	}
	if _, e = tx.Exec("DELETE FROM sessions"); e != nil {
		return e
	}
	return tx.Commit()
}
