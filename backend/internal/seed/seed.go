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
	SprintID        uint // the active sprint
	SprintsCreated  int
	IssuesCreated   int
	LabelsCreated   int
	CommentsCreated int
	// RelationsCreated counts the rows written; a "blocks" link is two rows.
	RelationsCreated int
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
		txRelation := repository.NewIssueRelationRepository(tx)

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
			Role:      domain.RoleAdmin,
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

		// Create three sprints: one completed, one in progress, one planned.
		// Dates are relative to today so the demo always looks current: the
		// completed sprint feeds velocity, the active one the burndown.
		today := time.Now().UTC().Truncate(24 * time.Hour)
		day := func(offset int) *time.Time {
			t := today.AddDate(0, 0, offset)
			return &t
		}
		sprintFixtures := []*domain.Sprint{
			{ProjectID: proj.ID, Name: "Sprint 1", Goal: "Lay the project foundations", Status: domain.SprintStatusCompleted, StartDate: day(-21), EndDate: day(-8)},
			{ProjectID: proj.ID, Name: "Sprint 2", Goal: "Ship the first set of demo features", Status: domain.SprintStatusActive, StartDate: day(-7), EndDate: day(6)},
			{ProjectID: proj.ID, Name: "Sprint 3", Goal: "Reporting and polish", Status: domain.SprintStatusPlanning, StartDate: day(7), EndDate: day(20)},
		}
		for _, sp := range sprintFixtures {
			if err := txSprint.Create(ctx, sp); err != nil {
				return err
			}
		}
		done, active, planned := sprintFixtures[0].ID, sprintFixtures[1].ID, sprintFixtures[2].ID
		result.SprintID = active
		result.SprintsCreated = len(sprintFixtures)

		type issueFixture struct {
			title    string
			typ      string
			status   string
			priority string
			points   int
			sprintID *uint
			// doneDaysAgo sets CompletedAt for done issues (burndown and velocity).
			doneDaysAgo int
			assigned    bool
		}
		fixtures := []issueFixture{
			// Sprint 1: completed, every ticket done.
			{"Set up project repository", domain.IssueTypeTask, domain.IssueStatusDone, domain.IssuePriorityLow, 2, &done, 19, true},
			{"Configure CI pipeline", domain.IssueTypeTask, domain.IssueStatusDone, domain.IssuePriorityMedium, 3, &done, 17, true},
			{"Design database schema", domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityHigh, 5, &done, 15, false},
			{"User registration and login", domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityHigh, 8, &done, 12, true},
			{"Fix typo in welcome email", domain.IssueTypeBug, domain.IssueStatusDone, domain.IssuePriorityLow, 1, &done, 10, false},
			{"Project list page", domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityMedium, 5, &done, 9, true},
			// Sprint 2: in progress, spread over every board column.
			{"Design system colour tokens", domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityMedium, 3, &active, 4, true},
			{"Add issue comments", domain.IssueTypeStory, domain.IssueStatusDone, domain.IssuePriorityMedium, 5, &active, 2, false},
			{"Fix login page redirect loop", domain.IssueTypeBug, domain.IssueStatusInReview, domain.IssuePriorityHigh, 3, &active, 0, true},
			{"Implement Kanban board drag-and-drop", domain.IssueTypeStory, domain.IssueStatusInProgress, domain.IssuePriorityHigh, 8, &active, 0, true},
			{"Write API integration tests", domain.IssueTypeTask, domain.IssueStatusInProgress, domain.IssuePriorityMedium, 5, &active, 0, false},
			{"Email notifications for assignments", domain.IssueTypeStory, domain.IssueStatusTodo, domain.IssuePriorityMedium, 5, &active, 0, true},
			{"Search is slow on large projects", domain.IssueTypeBug, domain.IssueStatusTodo, domain.IssuePriorityCritical, 2, &active, 0, false},
			// Sprint 3: planned, not started.
			{"Burndown and velocity reports", domain.IssueTypeStory, domain.IssueStatusTodo, domain.IssuePriorityMedium, 5, &planned, 0, false},
			{"Add CSV export for sprint reports", domain.IssueTypeTask, domain.IssueStatusTodo, domain.IssuePriorityLow, 3, &planned, 0, false},
			{"Dark mode contrast fixes", domain.IssueTypeBug, domain.IssueStatusTodo, domain.IssuePriorityMedium, 2, &planned, 0, false},
			{"Keyboard shortcuts for the board", domain.IssueTypeStory, domain.IssueStatusTodo, domain.IssuePriorityLow, 5, &planned, 0, false},
			{"Document the REST API", domain.IssueTypeTask, domain.IssueStatusTodo, domain.IssuePriorityMedium, 3, &planned, 0, false},
			// Backlog: not in any sprint.
			{"Performance regression on issue list", domain.IssueTypeBug, domain.IssueStatusBacklog, domain.IssuePriorityCritical, 5, nil, 0, false},
			{"Saved filters", domain.IssueTypeStory, domain.IssueStatusBacklog, domain.IssuePriorityLow, 8, nil, 0, false},
		}

		var issues []*domain.Issue
		for i, f := range fixtures {
			points := f.points
			issue := &domain.Issue{
				ProjectID:   proj.ID,
				Title:       f.title,
				Description: loremDescription(i, f.typ),
				Type:        f.typ,
				Status:      f.status,
				Priority:    f.priority,
				StoryPoints: &points,
				SprintID:    f.sprintID,
				ReporterID:  adminID,
			}
			if f.assigned {
				assignee := adminID
				issue.AssigneeID = &assignee
			}
			if f.status == domain.IssueStatusDone {
				completed := today.AddDate(0, 0, -f.doneDaysAgo).Add(12 * time.Hour)
				issue.CompletedAt = &completed
			}
			if err := txIssue.Create(ctx, issue); err != nil {
				return err
			}
			issues = append(issues, issue)
		}
		result.IssuesCreated = len(issues)

		// Attach labels: feature→colour tokens, bug→login redirect loop, feature→kanban board
		if err := txLabel.AttachToIssue(ctx, issues[6].ID, featureLabel.ID); err != nil {
			return err
		}
		if err := txLabel.AttachToIssue(ctx, issues[8].ID, bugLabel.ID); err != nil {
			return err
		}
		if err := txLabel.AttachToIssue(ctx, issues[9].ID, featureLabel.ID); err != nil {
			return err
		}

		// Create comments. Two tickets keep a hand-written opening comment; every
		// ticket gets a short conversation of placeholder text, and a few have long
		// threads (more than ten comments) so the comment list can be seen at length.
		openers := map[int]string{
			8: "Reproduced on Firefox and Safari. Likely the redirect guard in auth.global.ts.",
			9: "Considering Vue-draggable-plus for the board component.",
		}
		longThreads := map[int]int{3: 12, 8: 11, 9: 14, 12: 10}
		shortThreads := []int{1, 2, 3, 2}
		for idx, issue := range issues {
			count, long := longThreads[idx]
			if !long {
				count = shortThreads[idx%len(shortThreads)]
			}
			for n := 0; n < count; n++ {
				body := loremComment(idx, n)
				if opener, ok := openers[idx]; ok && n == 0 {
					body = opener
				}
				if err := txComment.Create(ctx, &domain.Comment{IssueID: issue.ID, AuthorID: adminID, Body: body}); err != nil {
					return err
				}
				result.CommentsCreated++
			}
		}

		// Link related tickets (indexes are positions in fixtures above). A "blocks"
		// link is stored as two rows, like the relation service does it.
		type link struct {
			from, to int
			kind     string
		}
		links := []link{
			{9, 16, domain.RelationTypeBlocks},      // Kanban board drag-and-drop -> Keyboard shortcuts for the board
			{7, 11, domain.RelationTypeBlocks},      // Add issue comments -> Email notifications for assignments
			{2, 3, domain.RelationTypeBlocks},       // Design database schema -> User registration and login
			{12, 19, domain.RelationTypeBlocks},     // Search is slow on large projects -> Saved filters
			{8, 3, domain.RelationTypeRelatesTo},    // Login redirect loop <-> User registration and login
			{6, 15, domain.RelationTypeRelatesTo},   // Colour tokens <-> Dark mode contrast fixes
			{13, 14, domain.RelationTypeRelatesTo},  // Burndown reports <-> CSV export for sprint reports
			{10, 17, domain.RelationTypeRelatesTo},  // API integration tests <-> Document the REST API
			{18, 12, domain.RelationTypeDuplicates}, // Performance regression on issue list duplicates Search is slow
		}
		for _, l := range links {
			rows := []*domain.IssueRelation{{IssueID: issues[l.from].ID, RelatedIssueID: issues[l.to].ID, RelationType: l.kind, CreatedByID: adminID}}
			if l.kind == domain.RelationTypeBlocks {
				rows = append(rows, &domain.IssueRelation{IssueID: issues[l.to].ID, RelatedIssueID: issues[l.from].ID, RelationType: domain.RelationTypeIsBlockedBy, CreatedByID: adminID})
			}
			for _, row := range rows {
				if err := txRelation.Create(ctx, row); err != nil {
					return err
				}
				result.RelationsCreated++
			}
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}
