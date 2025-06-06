package feed

import (
	"context"
	"log"
)

func StartService(ctx context.Context) {
	log.Println("[Feed] Starting feed service...")

	go runCreator(ctx)
	go runResponder(ctx)
	go runManager(ctx)
	go runCleanup(ctx)
}
