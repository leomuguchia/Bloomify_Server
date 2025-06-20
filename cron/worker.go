package cron

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"bloomify/config"
	"bloomify/models"
	"bloomify/services/notification"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
)

const TypeReminderSend = "reminder:send"

// InitReminderWorker runs the async worker in background.
func InitReminderWorker(notifSvc notification.NotificationService) {
	redisOpts := asynq.RedisClientOpt{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisReminderQueueDB,
	}

	srv := asynq.NewServer(
		redisOpts,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"default": 1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeReminderSend, handleReminderTask(notifSvc))

	// Start Redis health monitor
	go monitorRedisConnection()

	// Start async worker with retry logic
	go func() {
		log.Println("[ReminderWorker] 🚀 Starting async worker...")
		const maxAttempts = 5

		for attempts := 1; attempts <= maxAttempts; attempts++ {
			if err := srv.Run(mux); err != nil {
				log.Printf("[ReminderWorker] ❌ Attempt %d/%d failed to start worker: %v", attempts, maxAttempts, err)

				if attempts == maxAttempts {
					log.Fatal("[ReminderWorker] ❗ Max retry attempts reached. Exiting.")
				}
				time.Sleep(time.Duration(attempts*2) * time.Second) // Exponential backoff
			} else {
				break
			}
		}
	}()
}

func handleReminderTask(notifSvc notification.NotificationService) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		var p models.ReminderPayload
		if err := json.Unmarshal(task.Payload(), &p); err != nil {
			log.Printf("[ReminderHandler] 🔴 Invalid payload: %v", err)
			return err
		}

		log.Printf("[ReminderHandler] ⏰ Triggering reminder for %s %s → %s: %s", p.Target, p.ID, p.Title, p.Body)

		data := map[string]string{
			"reminderId": p.ReminderID,
			"fireDate":   p.FireDate,
			"title":      p.Title,
			"body":       p.Body,
		}

		var err error
		switch p.Target {
		case "user":
			err = notifSvc.SendUserPushNotification(ctx, p.ID, p.Title, p.Body, data)
		case "provider":
			err = notifSvc.SendProviderPushNotification(ctx, p.ID, p.Title, p.Body, data)
		default:
			log.Printf("[ReminderHandler] ⚠️ Unknown target type: %s", p.Target)
			return nil
		}

		if err != nil {
			log.Printf("[ReminderHandler] ❌ Failed to send notification: %v", err)
		}
		return err
	}
}

// monitorRedisConnection pings Redis periodically to detect failures at runtime.
func monitorRedisConnection() {
	client := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisReminderQueueDB,
	})

	ctx := context.Background()

	for {
		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("[ReminderWorker] ⚠️ Redis connection lost: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}
