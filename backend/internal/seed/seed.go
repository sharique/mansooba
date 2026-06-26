package seed

import (
	"context"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"gorm.io/gorm"
)

// SeedResult is returned by Seed on success.
type SeedResult struct {
	Skipped         bool
	ProjectKey      string
	ProjectName     string
	ProjectID       uint
	SprintID        uint
	IssuesCreated   int
	LabelsCreated   int
	CommentsCreated int
}

// Seed populates the DB with demo data inside a single GORM transaction.
// Returns SeedResult{Skipped: true} if seed data already exists (idempotent).
// Returns an error (with zero inserts) if adminID does not refer to a real user.
func Seed(ctx context.Context, db *gorm.DB, adminID uint) (*SeedResult, error) {
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	// Verify admin exists before opening any transaction.
	if _, err := userRepo.FindByID(ctx, adminID); err != nil {
		return nil, err
	}

	// Idempotency check: skip only when the seed project itself already exists.
	// Distinguish seed-created projects by name; a user-created "DEMO" project
	// has a different name and triggers the SDEMO conflict fallback instead.
	if p, err := projectRepo.FindByKey(ctx, "DEMO"); err == nil && p.Name == "Mansooba Demo" {
		return &SeedResult{Skipped: true, ProjectKey: "DEMO", ProjectName: "Mansooba Demo"}, nil
	}
	if p, err := projectRepo.FindByKey(ctx, "SDEMO"); err == nil && p.Name == "Seed Demo Project" {
		return &SeedResult{Skipped: true, ProjectKey: "SDEMO", ProjectName: "Seed Demo Project"}, nil
	}

	result := &SeedResult{}

	txErr := db.Transaction(func(tx *gorm.DB) error {
		txProject := repository.NewProjectRepository(tx)
		txMember := repository.NewProjectMemberRepository(tx)
		txLabel := repository.NewLabelRepository(tx)
		txSprint := repository.NewSprintRepository(tx)
		txIssue := repository.NewIssueRepository(tx)
		txComment := repository.NewCommentRepository(tx)

		// Determine key: use DEMO unless already taken (inside tx, race-safe).
		key := "DEMO"
		name := "Mansooba Demo"
		if _, err := txProject.FindByKey(ctx, "DEMO"); err == nil {
			key = "SDEMO"
			name = "Seed Demo Project"
		}
		result.ProjectKey = key
		result.ProjectName = name

		// Create project.
		proj := &domain.Project{
			Key:         key,
			Name:        name,
			Description: "A sample project to explore Mansooba's features",
			OwnerID:     adminID,
		}
		if err := txProject.Create(ctx, proj); err != nil {
			return err
		}
		result.ProjectID = proj.ID

		// Add admin as project admin member.
		if err := txMember.Create(ctx, &domain.ProjectMember{
			ProjectID: proj.ID,
			UserID:    adminID,
			Role:      "admin",
		}); err != nil {
			return err
		}

		// Create labels.
		bugLabel := &domain.Label{ProjectID: proj.ID, Name: "bug", Color: "#e11d48"}
		featureLabel := &domain.Label{ProjectID: proj.ID, Name: "feature", Color: "#3b82f6"}
		for _, lbl := range []*domain.Label{bugLabel, featureLabel} {
			if err := txLabel.Create(ctx, lbl); err != nil {
				return err
			}
		}
		result.LabelsCreated = 2

		// Create sprint.
		now := time.Now().UTC().Truncate(24 * time.Hour)
		end := now.Add(14 * 24 * time.Hour)
		sprint := &domain.Sprint{
			ProjectID: proj.ID,
			Name:      "Sprint 1",
			Goal:      "Ship the first set of demo features",
			Status:    domain.SprintStatusActive,
			StartDate: &now,
			EndDate:   &end,
		}
		if err := txSprint.Create(ctx, sprint); err != nil {
			return err
		}
		result.SprintID = sprint.ID

		sprintID := sprint.ID

		type issueFixture struct {
			title    string
			typ      string
			status   string
			priority string
			sprintID *uint
		}
		fixtures := []issueFixture{
			{"Set up project repository", domain.IssueTypeTask, domain.IssueStatusDone, domain.IssuePriorityLow, &sprintID},
			{"Design system colour tokens", domain.IssueTypeStory, domain.IssueStatusInReview, domain.IssuePriorityMedium, &sprintID},
			{"Fix login page redirect loop", domain.IssueTypeBug, domain.IssueStatusInProgress, domain.IssuePriorityHigh, &sprintID},
			{"Implement Kanban board drag-and-drop", domain.IssueTypeStory, domain.IssueStatusTodo, domain.IssuePriorityMedium, nil},
			{"Write API integration tests", domain.IssueTypeTask, domain.IssueStatusTodo, domain.IssuePriorityMedium, nil},
			{"Performance regression on issue list", domain.IssueTypeBug, domain.IssueStatusBacklog, domain.IssuePriorityCritical, nil},
			{"Add CSV export for sprint reports", domain.IssueTypeTask, domain.IssueStatusBacklog, domain.IssuePriorityLow, nil},
		}

		var issues []*domain.Issue
		for i, f := range fixtures {
			issue := &domain.Issue{
				Key:        key + "-" + itoa(i+1),
				ProjectID:  proj.ID,
				Title:      f.title,
				Type:       f.typ,
				Status:     f.status,
				Priority:   f.priority,
				SprintID:   f.sprintID,
				ReporterID: adminID,
			}
			if err := txIssue.Create(ctx, issue); err != nil {
				return err
			}
			issues = append(issues, issue)
		}
		result.IssuesCreated = len(issues)

		// Attach labels: bug→issue[2] (login bug), feature→issue[1] (colour tokens), feature→issue[3] (kanban)
		if err := txLabel.AttachToIssue(ctx, issues[1].ID, featureLabel.ID); err != nil {
			return err
		}
		if err := txLabel.AttachToIssue(ctx, issues[2].ID, bugLabel.ID); err != nil {
			return err
		}
		if err := txLabel.AttachToIssue(ctx, issues[3].ID, featureLabel.ID); err != nil {
			return err
		}

		// Create comments.
		comments := []struct {
			issueIdx int
			body     string
		}{
			{2, "Reproduced on Firefox and Safari. Likely the redirect guard in auth.global.ts."},
			{3, "Considering Vue-draggable-plus for the board component."},
		}
		for _, c := range comments {
			if err := txComment.Create(ctx, &domain.Comment{
				IssueID:  issues[c.issueIdx].ID,
				AuthorID: adminID,
				Body:     c.body,
			}); err != nil {
				return err
			}
		}
		result.CommentsCreated = 2

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := []byte{}
	if n == 0 {
		return "0"
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
