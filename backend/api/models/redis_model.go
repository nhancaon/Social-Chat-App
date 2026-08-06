package models

// chachedrespose for all posts
type CachedGetAllPostResponse struct {
	Data          []PostModel `json:"data"`
	CurrentPage   int         `json:"currentPage"`
	NumberOfPages float64     `json:"numberOfPages"`
}

// cachedrespose for user profile
type CachedGetUserResponse struct {
	User          UserModel   `json:"user"`
	Posts         []PostModel `json:"posts"`
	CurrentPage   int         `json:"currentPage"`
	NumberOfPages float64     `json:"numberOfPages"`
}
