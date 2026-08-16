package models

// cached response for all posts
type CachedGetAllPostResponse struct {
	Data          []PostModel `json:"data"`
	CurrentPage   int         `json:"currentPage"`
	NumberOfPages float64     `json:"numberOfPages"`
}

// cached response for user profile
type CachedGetUserResponse struct {
	User          UserModel   `json:"user"`
	Posts         []PostModel `json:"posts"`
	CurrentPage   int         `json:"currentPage"`
	NumberOfPages float64     `json:"numberOfPages"`
}
