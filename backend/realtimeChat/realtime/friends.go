package realtime

import (
	"log"
	"realTimeChat/gapi"
)

func GetUserFriends(userID string) <-chan []string {
	ch := make(chan []string)
	go func() {
		defer close(ch)
		userFriends, err := gapi.GetFollowingFollowersClient(userID)
		if err != nil {
			log.Printf("GetUserFriends: error calling grpc for user %s: %v", userID, err)
			ch <- []string{}
			return
		}

		friends := []string{} // explicit empty slice, not nil, if there are no friends
		for _, userIDsList := range userFriends {
			friends = append(friends, userIDsList.UserIdsList...)
		}
		ch <- friends
	}()
	return ch
}
