package controllers

import (
	"Server/kafka"
	"Server/models"
	"Server/realtime"
	"log"
)

// publishNotification pushes a saved notification through the realtime
// (Kafka-backed) delivery path. Best-effort: failures are logged, not
// returned, since notification delivery must never fail the request that
// triggered it.
func publishNotification(notification models.Notification) {
	notificationHub := realtime.GetNotificationHub()
	if notificationHub == nil {
		log.Printf("notification hub not ready, dropping realtime notification for user %s", notification.MainUID)
		return
	}

	payload := kafka.Notification{
		ID:        notification.ID.Hex(),
		Details:   notification.Details,
		MainUID:   notification.MainUID,
		TargetID:  notification.TargetID,
		UserID:    notification.UserID,
		IsRead:    notification.IsRead,
		CreatedAt: notification.CreatedAt,
		User: kafka.User{
			Name:   notification.User.Name,
			Avatar: notification.User.Avatar,
		},
	}

	if err := notificationHub.SendNotificatonToUser(notification.MainUID, payload); err != nil {
		log.Printf("failed to publish realtime notification: %v", err)
	}
}
