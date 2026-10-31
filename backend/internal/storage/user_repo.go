package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"comics-backend/internal/domain"
)

type UserRepository struct {
	db       *sql.DB
	memMu    sync.RWMutex
	memUsers []domain.User
	jsonPath string
}

func NewUserRepository(db *sql.DB, dataDir ...string) *UserRepository {
	dir := "storage"
	if len(dataDir) > 0 && dataDir[0] != "" {
		dir = dataDir[0]
	}
	repo := &UserRepository{
		db:       db,
		jsonPath: filepath.Join(dir, "users.json"),
		memUsers: make([]domain.User, 0),
	}
	repo.loadFromJSON()
	if len(repo.memUsers) == 0 {
		repo.seedDefaultUser()
	}
	return repo
}

func (r *UserRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

func hashPassword(pwd string) string {
	hash := sha256.Sum256([]byte(pwd + "_tu_tien_secret_2026"))
	return hex.EncodeToString(hash[:])
}

func (r *UserRepository) seedDefaultUser() {
	r.memMu.Lock()
	defer r.memMu.Unlock()
	defaultUser := domain.User{
		ID:        "usr-1",
		Username:  "TieuViem",
		Email:     "tieuviem@tu-tien.vn",
		Password:  hashPassword("123456"),
		Realm:     "Kim Đan Kỳ",
		Exp:       2800,
		AvatarURL: "https://api.dicebear.com/7.x/bottts/svg?seed=TieuViem",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.memUsers = append(r.memUsers, defaultUser)
	r.saveToJSONLocked()
}

func (r *UserRepository) loadFromJSON() {
	data, err := os.ReadFile(r.jsonPath)
	if err != nil {
		return
	}
	var users []domain.User
	if err := json.Unmarshal(data, &users); err == nil {
		r.memUsers = users
	}
}

func (r *UserRepository) saveToJSONLocked() {
	_ = os.MkdirAll(filepath.Dir(r.jsonPath), 0755)
	data, err := json.MarshalIndent(r.memUsers, "", "  ")
	if err == nil {
		_ = os.WriteFile(r.jsonPath, data, 0644)
	}
}

// Register creates a new user
func (r *UserRepository) Register(username, email, password string) (*domain.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	if username == "" || email == "" || password == "" {
		return nil, errors.New("vui lòng điền đầy đủ đạo hiệu, email và mật khẩu")
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()

	for _, u := range r.memUsers {
		if strings.EqualFold(u.Username, username) {
			return nil, errors.New("đạo hiệu này đã có người đăng ký")
		}
		if strings.EqualFold(u.Email, email) {
			return nil, errors.New("email này đã được sử dụng")
		}
	}

	newUser := domain.User{
		ID:        fmt.Sprintf("usr-%d", time.Now().UnixNano()%1000000),
		Username:  username,
		Email:     email,
		Password:  hashPassword(password),
		Realm:     "Phàm Nhân",
		Exp:       10,
		AvatarURL: fmt.Sprintf("https://api.dicebear.com/7.x/bottts/svg?seed=%s", username),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if r.dbAvailable() {
		_, _ = r.db.Exec(`
			INSERT INTO users (id, username, email, password_hash, realm, exp, avatar_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, newUser.ID, newUser.Username, newUser.Email, newUser.Password, newUser.Realm, newUser.Exp, newUser.AvatarURL, newUser.CreatedAt, newUser.UpdatedAt)
	}

	r.memUsers = append(r.memUsers, newUser)
	r.saveToJSONLocked()
	return &newUser, nil
}

// Authenticate verifies credentials
func (r *UserRepository) Authenticate(login, password string) (*domain.User, error) {
	login = strings.TrimSpace(login)
	pwdHash := hashPassword(password)

	r.memMu.RLock()
	defer r.memMu.RUnlock()

	for _, u := range r.memUsers {
		if (strings.EqualFold(u.Username, login) || strings.EqualFold(u.Email, login)) && u.Password == pwdHash {
			userCopy := u
			userCopy.Password = ""
			return &userCopy, nil
		}
	}
	return nil, errors.New("đạo hiệu hoặc mật khẩu không chính xác")
}

// GetByID returns user profile
func (r *UserRepository) GetByID(id string) (*domain.User, error) {
	r.memMu.RLock()
	defer r.memMu.RUnlock()

	for _, u := range r.memUsers {
		if u.ID == id {
			userCopy := u
			userCopy.Password = ""
			return &userCopy, nil
		}
	}
	return nil, errors.New("không tìm thấy người dùng")
}

// AddExp increases cultivation experience when reading chapters
func (r *UserRepository) AddExp(id string, points int) (*domain.User, error) {
	r.memMu.Lock()
	defer r.memMu.Unlock()

	for i, u := range r.memUsers {
		if u.ID == id {
			r.memUsers[i].Exp += points
			newRealm, _ := domain.CalculateRealm(r.memUsers[i].Exp)
			r.memUsers[i].Realm = newRealm
			r.memUsers[i].UpdatedAt = time.Now()

			if r.dbAvailable() {
				_, _ = r.db.Exec(`
					UPDATE users
					SET exp = $1, realm = $2, updated_at = NOW()
					WHERE id = $3
				`, r.memUsers[i].Exp, newRealm, id)
			}

			r.saveToJSONLocked()
			userCopy := r.memUsers[i]
			userCopy.Password = ""
			return &userCopy, nil
		}
	}
	return nil, errors.New("không tìm thấy người dùng")
}

