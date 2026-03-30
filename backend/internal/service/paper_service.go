package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	pbgrpc "github.com/khalily/oreader/internal/infra/grpc"
	"github.com/khalily/oreader/internal/infra/logger"
	"github.com/khalily/oreader/internal/model"
)

const defaultConversionTimeout = 10 * time.Minute

type paperService struct {
	paperRepo     PaperRepository
	tagRepo       PaperTagRepository
	grpcClient    pbgrpc.PaperConverterClient
	uploadDir     string
	grpcTimeout   time.Duration
	conversionMu  sync.Mutex // B4: prevent concurrent updates on same paper
}

// NewPaperService creates a new paper service
func NewPaperService(
	paperRepo PaperRepository,
	tagRepo PaperTagRepository,
	grpcClient pbgrpc.PaperConverterClient,
	uploadDir string,
) PaperService {
	return &paperService{
		paperRepo:   paperRepo,
		tagRepo:     tagRepo,
		grpcClient:  grpcClient,
		uploadDir:   uploadDir,
		grpcTimeout: defaultConversionTimeout,
	}
}

// UploadPaper saves the PDF to disk, creates a Paper record, and starts async conversion
func (s *paperService) UploadPaper(ctx context.Context, userID string, filename string, pdfContent []byte) (*model.Paper, error) {
	// B3: Check if gRPC converter is available
	if s.grpcClient == nil {
		return nil, fmt.Errorf("paper converter service is not available, please try again later")
	}

	// S4: Sanitize filename to prevent path traversal
	safeFilename := filepath.Base(filename)
	if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
		return nil, fmt.Errorf("invalid filename")
	}

	// Save PDF to disk organized by user and date
	dateDir := time.Now().Format("2006-01-02")
	pdfDir := filepath.Join(s.uploadDir, userID, dateDir)
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	pdfPath := filepath.Join(pdfDir, safeFilename)
	if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to save PDF: %w", err)
	}

	// Create Paper record in pending status
	paper := &model.Paper{
		UserID:           userID,
		OriginalFilename: safeFilename,
		PDFPath:          pdfPath,
		PDFSize:          int64(len(pdfContent)),
		Status:           model.PaperStatusPending,
	}
	if err := paper.GenerateID(); err != nil {
		return nil, fmt.Errorf("failed to generate paper ID: %w", err)
	}

	if err := s.paperRepo.Create(ctx, paper); err != nil {
		// Clean up saved file on DB error
		os.Remove(pdfPath)
		return nil, fmt.Errorf("failed to create paper record: %w", err)
	}

	logger.Info().
		Str("paper_id", paper.ID).
		Str("user_id", userID).
		Str("filename", safeFilename).
		Int64("size", paper.PDFSize).
		Msg("Paper uploaded, starting conversion")

	// P1: Don't pass pdfContent to goroutine — read from disk instead
	go s.processConversion(paper.ID, safeFilename)

	return paper, nil
}

// processConversion runs in a goroutine to convert PDF and extract metadata
func (s *paperService) processConversion(paperID string, filename string) {
	// P3: Use a context with timeout for all RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), s.grpcTimeout)
	defer cancel()

	// B4: Lock to prevent concurrent conversion updates on same paper
	s.conversionMu.Lock()
	defer s.conversionMu.Unlock()

	// Update status to processing
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to get paper for conversion")
		return
	}
	if paper == nil {
		logger.Error().Str("paper_id", paperID).Msg("Paper not found for conversion")
		return
	}

	paper.Status = model.PaperStatusProcessing
	if err := s.paperRepo.Update(ctx, paper); err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to update paper status to processing")
		return
	}

	// P1: Read PDF from disk instead of holding bytes in memory
	pdfContent, err := os.ReadFile(paper.PDFPath)
	if err != nil {
		s.failPaper(ctx, paperID, fmt.Sprintf("failed to read PDF file: %v", err))
		return
	}

	// Call gRPC Convert (streaming)
	progresses, err := s.grpcClient.Convert(ctx, pdfContent, filename)
	if err != nil {
		s.failPaper(ctx, paperID, fmt.Sprintf("conversion failed: %v", err))
		return
	}

	// Get the final markdown from the last progress update
	var markdown string
	for _, p := range progresses {
		if p.Markdown != "" {
			markdown = p.Markdown
		}
		if p.Error != "" {
			s.failPaper(ctx, paperID, fmt.Sprintf("conversion error: %s", p.Error))
			return
		}
	}

	if markdown == "" {
		s.failPaper(ctx, paperID, "conversion returned empty markdown")
		return
	}

	// Extract metadata
	metadata, err := s.grpcClient.ExtractMetadata(ctx, markdown)
	if err != nil {
		logger.Warn().Err(err).Str("paper_id", paperID).Msg("Metadata extraction failed, saving markdown without metadata")
		// Save markdown without metadata
		paper.MarkdownContent = markdown
		paper.Status = model.PaperStatusCompleted
		paper.Title = paper.OriginalFilename
		if err := s.paperRepo.Update(ctx, paper); err != nil {
			logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to update paper after metadata extraction failure")
		}
		return
	}

	// Update paper with conversion results
	authorsJSON, _ := json.Marshal(metadata.Authors)
	keywordsJSON, _ := json.Marshal(metadata.Keywords)

	paper.MarkdownContent = markdown
	paper.Status = model.PaperStatusCompleted
	paper.Title = metadata.Title
	paper.Authors = string(authorsJSON)
	paper.Abstract = metadata.Abstract
	paper.Keywords = string(keywordsJSON)
	paper.PublishedYear = metadata.PublishedYear
	paper.DOI = metadata.DOI

	if err := s.paperRepo.Update(ctx, paper); err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to update paper with conversion results")
		return
	}

	logger.Info().
		Str("paper_id", paperID).
		Str("title", paper.Title).
		Msg("Paper conversion completed")
}

// failPaper marks a paper as failed with the given error message
func (s *paperService) failPaper(ctx context.Context, paperID string, errMsg string) {
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil || paper == nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to get paper for marking as failed")
		return
	}

	paper.Status = model.PaperStatusFailed
	paper.Error = errMsg
	if err := s.paperRepo.Update(ctx, paper); err != nil {
		logger.Error().Err(err).Str("paper_id", paperID).Msg("Failed to mark paper as failed")
	}

	logger.Error().
		Str("paper_id", paperID).
		Str("error", errMsg).
		Msg("Paper conversion failed")
}

// GetPaper retrieves a paper with ownership check
func (s *paperService) GetPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	paper, err := s.paperRepo.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil || paper.UserID != userID {
		return nil, ErrPaperNotFound
	}
	return paper, nil
}

// ListPapers retrieves papers for a user with filtering and pagination
func (s *paperService) ListPapers(ctx context.Context, userID string, opts PaperListOptions) ([]*model.Paper, int64, error) {
	return s.paperRepo.ListByUserID(ctx, userID, opts)
}

// UpdatePaper updates a paper's fields with ownership check
func (s *paperService) UpdatePaper(ctx context.Context, userID, paperID string, updates map[string]interface{}) (*model.Paper, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	// Apply allowed updates
	if title, ok := updates["title"].(string); ok {
		paper.Title = title
	}
	if abstract, ok := updates["abstract"].(string); ok {
		paper.Abstract = abstract
	}
	if publishedYear, ok := updates["published_year"].(string); ok {
		paper.PublishedYear = publishedYear
	}
	if doi, ok := updates["doi"].(string); ok {
		paper.DOI = doi
	}

	if err := s.paperRepo.Update(ctx, paper); err != nil {
		return nil, err
	}

	return paper, nil
}

// DeletePaper deletes a paper with ownership check, also removes the PDF file
func (s *paperService) DeletePaper(ctx context.Context, userID, paperID string) error {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return err
	}

	// Delete PDF file from disk
	if paper.PDFPath != "" {
		if err := os.Remove(paper.PDFPath); err != nil && !os.IsNotExist(err) {
			logger.Warn().Err(err).Str("paper_id", paperID).Str("path", paper.PDFPath).Msg("Failed to delete PDF file")
		}
	}

	return s.paperRepo.Delete(ctx, paperID)
}

// GetPaperStatus retrieves the conversion status of a paper
func (s *paperService) GetPaperStatus(ctx context.Context, userID, paperID string) (*PaperStatusResponse, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	var progress int32
	switch paper.Status {
	case model.PaperStatusPending:
		progress = 0
	case model.PaperStatusProcessing:
		progress = 50
	case model.PaperStatusCompleted:
		progress = 100
	case model.PaperStatusFailed:
		progress = 0
	}

	return &PaperStatusResponse{
		ID:        paper.ID,
		Status:    paper.Status,
		Progress:  progress,
		Error:     paper.Error,
		CreatedAt: paper.CreatedAt.Format(time.RFC3339),
		UpdatedAt: paper.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// RetryPaper retries conversion for a failed paper
func (s *paperService) RetryPaper(ctx context.Context, userID, paperID string) (*model.Paper, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return nil, err
	}

	if paper.Status != model.PaperStatusFailed {
		return nil, fmt.Errorf("can only retry failed papers, current status: %s", paper.Status)
	}

	// Reset paper status
	paper.Status = model.PaperStatusPending
	paper.Error = ""
	if err := s.paperRepo.Update(ctx, paper); err != nil {
		return nil, fmt.Errorf("failed to reset paper status: %w", err)
	}

	// Start async conversion (processConversion reads PDF from disk)
	go s.processConversion(paper.ID, paper.OriginalFilename)

	return paper, nil
}

// ListTags retrieves all tags for a user's papers
func (s *paperService) ListTags(ctx context.Context, userID string) ([]string, error) {
	return s.paperRepo.ListTags(ctx, userID)
}

// UpdateTags sets the tags for a paper with ownership check
func (s *paperService) UpdateTags(ctx context.Context, userID, paperID string, tags []string) error {
	// Ownership check
	if _, err := s.GetPaper(ctx, userID, paperID); err != nil {
		return err
	}

	return s.tagRepo.SetTags(ctx, paperID, tags)
}

// DownloadPaper returns the PDF file path and filename for download with ownership check
func (s *paperService) DownloadPaper(ctx context.Context, userID, paperID string) (string, string, error) {
	paper, err := s.GetPaper(ctx, userID, paperID)
	if err != nil {
		return "", "", err
	}

	// Extract filename from PDFPath
	filename := paper.OriginalFilename
	if filename == "" {
		filename = filepath.Base(paper.PDFPath)
	}

	// Verify file exists
	if _, err := os.Stat(paper.PDFPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("PDF file not found: %w", err)
		}
		return "", "", fmt.Errorf("failed to access PDF file: %w", err)
	}

	return paper.PDFPath, filename, nil
}
