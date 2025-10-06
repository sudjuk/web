package auth

import "sync"

var (
    currentUserIDOnce sync.Once
    currentUserID     int
)

// CurrentUserID returns a fixed creator ID per lab-3 requirement
func CurrentUserID() int {
    currentUserIDOnce.Do(func() {
        currentUserID = 1
    })
    return currentUserID
}






