package utils

import (
	"context"
	"log"
	"time"

	"bloomify/config"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
)

var (
	BookingCacheClient      *redis.Client
	AuthCacheClient         *redis.Client
	ProviderAuthCacheClient *redis.Client
	OTPCacheClient          *redis.Client
	TestCacheClient         *redis.Client
	AIContextCacheClient    *redis.Client
	FeedCacheClient         *redis.Client
	ReminderQueueClient     *asynq.Client
)

// --- Booking Cache ---
func InitBookingCache() {
	log.Printf("Attempting to connect to Redis (Booking Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisBookingCacheDB)
	BookingCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisBookingCacheDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := BookingCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Booking Cache): %v", err)
	}
	log.Println("Connected to Redis (Booking Cache) successfully.")
}
func GetBookingCacheClient() *redis.Client {
	if BookingCacheClient == nil {
		InitBookingCache()
	}
	return BookingCacheClient
}

// --- Auth Cache ---
func InitAuthCache() {
	log.Printf("Attempting to connect to Redis (Auth Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisAuthDB)
	AuthCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisAuthDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := AuthCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Auth Cache): %v", err)
	}
	log.Println("Connected to Redis (Auth Cache) successfully.")
}
func GetAuthCacheClient() *redis.Client {
	if AuthCacheClient == nil {
		InitAuthCache()
	}
	return AuthCacheClient
}

// --- Provider Auth Cache ---
const ProviderAuthCachePrefix = "auth:provider:"

func InitProviderAuthCache() {
	log.Printf("Attempting to connect to Redis (Provider Auth Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisProviderAuthDB)
	ProviderAuthCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisProviderAuthDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := ProviderAuthCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Provider Auth Cache): %v", err)
	}
	log.Println("Connected to Redis (Provider Auth Cache) successfully.")
}
func GetProviderAuthCacheClient() *redis.Client {
	if ProviderAuthCacheClient == nil {
		InitProviderAuthCache()
	}
	return ProviderAuthCacheClient
}

// --- OTP Cache ---
func InitOTPCache() {
	log.Printf("Attempting to connect to Redis (OTP Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisOTPDB)
	OTPCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisOTPDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := OTPCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (OTP Cache): %v", err)
	}
	log.Println("Connected to Redis (OTP Cache) successfully.")
}
func GetOTPCacheClient() *redis.Client {
	if OTPCacheClient == nil {
		InitOTPCache()
	}
	return OTPCacheClient
}

// --- AI Context Cache ---
func InitAIContextCache() {
	log.Printf("Attempting to connect to Redis (AI Context Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisAIContextDB)
	AIContextCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisAIContextDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := AIContextCacheClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis (AI Context Cache): %v", err)
	}
	log.Println("Connected to Redis (AI Context Cache) successfully.")
}
func GetAIContextCacheClient() *redis.Client {
	if AIContextCacheClient == nil {
		InitAIContextCache()
	}
	return AIContextCacheClient
}

// --- Feed Cache ---
const FeedCachePrefix = "feed:aggregates:"

func InitFeedCache() {
	log.Printf("Attempting to connect to Redis (Feed Cache) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisFeedDB)
	FeedCacheClient = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisFeedDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := FeedCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Feed Cache): %v", err)
	}
	log.Println("Connected to Redis (Feed Cache) successfully.")
}
func GetFeedCacheClient() *redis.Client {
	if FeedCacheClient == nil {
		InitFeedCache()
	}
	return FeedCacheClient
}

// --- Test Cache ---
func InitTestCache() {
	const (
		testAddr = "localhost:6379"
		testDB   = 5
	)
	log.Printf("Attempting to connect to Redis (Test Cache) at %s using DB %d", testAddr, testDB)
	TestCacheClient = redis.NewClient(&redis.Options{
		Addr:     testAddr,
		Password: "",
		DB:       testDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := TestCacheClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis (Test Cache): %v", err)
	}
	log.Println("Connected to Redis (Test Cache) successfully.")
}
func GetTestCacheClient() *redis.Client {
	if TestCacheClient == nil {
		InitTestCache()
	}
	return TestCacheClient
}

// --- Reminder Queue (Asynq) ---
func InitReminderQueue() {
	log.Printf("Initializing Redis Asynq Client (Reminder Queue) at %s using DB %d", config.AppConfig.RedisAddr, config.AppConfig.RedisReminderQueueDB)
	ReminderQueueClient = asynq.NewClient(asynq.RedisClientOpt{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       config.AppConfig.RedisReminderQueueDB,
	})
	log.Println("ReminderQueueClient initialized successfully.")
}
func GetReminderQueueClient() *asynq.Client {
	if ReminderQueueClient == nil {
		InitReminderQueue()
	}
	return ReminderQueueClient
}

// --- Redis Initialization ---
func InitRedis() {
	InitBookingCache()
	InitAuthCache()
	InitAIContextCache()
	InitOTPCache()
	InitFeedCache()
	InitProviderAuthCache()
	InitTestCache()
	InitReminderQueue()

	GetLogger().Sugar().Info("All Redis clients have been successfully initialized.")
}

// --- Aggregate Getter ---
func GetAllRedisClients() []*redis.Client {
	return []*redis.Client{
		GetBookingCacheClient(),
		GetAuthCacheClient(),
		GetProviderAuthCacheClient(),
		GetOTPCacheClient(),
		GetFeedCacheClient(),
		GetAIContextCacheClient(),
		GetTestCacheClient(),
	}
}
