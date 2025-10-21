package repository

import (
    "errors"
    "strings"

    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

type PublicUser struct {
    ID          int64  `json:"id"`
    Login       string `json:"login"`
    IsModerator bool   `json:"isModerator"`
}

func toPublic(u user) PublicUser { return PublicUser{ID: u.ID, Login: u.Login, IsModerator: u.IsModerator} }

func (r *Repository) Register(login, password string) (PublicUser, error) {
    login = strings.TrimSpace(login)
    if login == "" || password == "" { return PublicUser{}, errors.New("invalid input") }
    var exists int64
    if err := r.db.Model(&user{}).Where("login = ?", login).Count(&exists).Error; err != nil { return PublicUser{}, err }
    if exists > 0 { return PublicUser{}, errors.New("login taken") }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil { return PublicUser{}, err }
    u := user{Login: login, Password: string(hash), IsModerator: false}
    if err := r.db.Create(&u).Error; err != nil { return PublicUser{}, err }
    return toPublic(u), nil
}

func (r *Repository) Authenticate(login, password string) (PublicUser, error) {
    var u user
    if err := r.db.Where("login = ?", login).First(&u).Error; err != nil { return PublicUser{}, err }
    if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil { return PublicUser{}, errors.New("bad credentials") }
    return toPublic(u), nil
}

func (r *Repository) GetUserByID(id int64) (PublicUser, error) {
    var u user
    if err := r.db.Where("id = ?", id).First(&u).Error; err != nil { return PublicUser{}, err }
    return toPublic(u), nil
}

func (r *Repository) UpdateProfile(id int64, login *string, password *string) error {
    updates := map[string]interface{}{}
    if login != nil { v := strings.TrimSpace(*login); if v != "" { updates["login"] = v } }
    if password != nil { hash, _ := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost); updates["password_hash"] = string(hash) }
    if len(updates) == 0 { return nil }
    res := r.db.Model(&user{}).Where("id = ?", id).Updates(updates)
    if res.Error != nil { return res.Error }
    if res.RowsAffected == 0 { return gorm.ErrRecordNotFound }
    return nil
}






