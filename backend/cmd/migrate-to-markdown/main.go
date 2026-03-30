package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/infra/database"
	"github.com/khalily/oreader/internal/infra/markdown"
)

// Item represents a minimal item model for migration
type Item struct {
	ID      string
	Content string
}

func main() {
	// Parse flags
	dryRun := flag.Bool("dry-run", false, "Preview changes without modifying database")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	converter := markdown.NewConverter()

	log.Println("Starting HTML to Markdown migration...")
	if *dryRun {
		log.Println("DRY RUN MODE: No changes will be made to the database.")
	} else {
		log.Println("WARNING: This will modify all item content in the database.")
		log.Print("Continue? (y/N): ")

		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			log.Println("Migration cancelled.")
			os.Exit(0)
		}
	}

	// Count items to migrate
	var total int64
	db.Table("items").Count(&total)
	log.Printf("Found %d items to process", total)

	// Process in batches
	batchSize := 100
	offset := 0
	processed := 0
	skipped := 0
	failed := 0

	for {
		var items []Item
		result := db.Table("items").
			Select("id, content").
			Where("content IS NOT NULL AND content != ''").
			Offset(offset).
			Limit(batchSize).
			Find(&items)

		if result.Error != nil {
			log.Printf("Error fetching items: %v", result.Error)
			break
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			// Check if already Markdown (simple heuristic)
			if strings.Contains(item.Content, "```") ||
				strings.HasPrefix(item.Content, "#") {
				skipped++
				continue
			}

			mdContent, err := converter.Convert(item.Content)
			if err != nil {
				log.Printf("Failed to convert item %s: %v", item.ID, err)
				failed++
				continue
			}

			if *dryRun {
				log.Printf("[DRY-RUN] Would update item %s (content length: %d -> %d)",
					item.ID, len(item.Content), len(mdContent))
				processed++
			} else {
				result := db.Table("items").Where("id = ?", item.ID).Update("content", mdContent)
				if result.Error != nil {
					log.Printf("Failed to update item %s: %v", item.ID, result.Error)
					failed++
					continue
				}
				processed++
			}
		}

		offset += batchSize
		log.Printf("Processed %d, Skipped %d, Failed %d / %d items...",
			processed, skipped, failed, total)
	}

	log.Printf("Migration complete. Processed: %d, Skipped: %d, Failed: %d",
		processed, skipped, failed)

	if *dryRun {
		log.Println("This was a DRY RUN. No changes were made to the database.")
		log.Println("Run without --dry-run to apply changes.")
	}
}
