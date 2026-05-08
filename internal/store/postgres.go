package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sriniketh24/rollout/internal/models"
)

// Store provides access to the Postgres database for all Rollout entities.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a new Store backed by the given connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

func (s *Store) CreateProject(ctx context.Context, p *models.Project) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO projects (name, key) VALUES ($1, $2)
		 RETURNING id, created_at, updated_at`,
		p.Name, p.Key,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (s *Store) GetProject(ctx context.Context, id string) (*models.Project, error) {
	p := &models.Project{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, key, created_at, updated_at FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Key, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "project", id)
	}
	return p, nil
}

func (s *Store) GetProjectByKey(ctx context.Context, key string) (*models.Project, error) {
	p := &models.Project{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, key, created_at, updated_at FROM projects WHERE key = $1`, key,
	).Scan(&p.ID, &p.Name, &p.Key, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "project", key)
	}
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, key, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Key, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdateProject(ctx context.Context, p *models.Project) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE projects SET name = $1, key = $2, updated_at = now() WHERE id = $3`,
		p.Name, p.Key, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "project", ID: p.ID}
	}
	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "project", ID: id}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Environments
// ---------------------------------------------------------------------------

func (s *Store) CreateEnvironment(ctx context.Context, e *models.Environment) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO environments (project_id, name, key, color)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		e.ProjectID, e.Name, e.Key, e.Color,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (s *Store) GetEnvironment(ctx context.Context, id string) (*models.Environment, error) {
	e := &models.Environment{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, name, key, color, created_at, updated_at
		 FROM environments WHERE id = $1`, id,
	).Scan(&e.ID, &e.ProjectID, &e.Name, &e.Key, &e.Color, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "environment", id)
	}
	return e, nil
}

func (s *Store) ListEnvironments(ctx context.Context, projectID string) ([]models.Environment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, project_id, name, key, color, created_at, updated_at
		 FROM environments WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	defer rows.Close()

	var out []models.Environment
	for rows.Next() {
		var e models.Environment
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Key, &e.Color, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan environment: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UpdateEnvironment(ctx context.Context, e *models.Environment) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE environments SET name = $1, key = $2, color = $3, updated_at = now()
		 WHERE id = $4`,
		e.Name, e.Key, e.Color, e.ID,
	)
	if err != nil {
		return fmt.Errorf("update environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "environment", ID: e.ID}
	}
	return nil
}

func (s *Store) DeleteEnvironment(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "environment", ID: id}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

func (s *Store) CreateFlag(ctx context.Context, f *models.Flag) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO flags (project_id, key, name, description, type, default_value,
		                     enabled, tags, depends_on, kill_switch, archived, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		f.ProjectID, f.Key, f.Name, f.Description, f.Type, f.DefaultValue,
		f.Enabled, f.Tags, f.DependsOn, f.KillSwitch, f.Archived, f.CreatedBy,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

// GetFlag retrieves a flag by project ID and flag key.
// This satisfies the evaluator FlagStore interface.
func (s *Store) GetFlag(projectID, flagKey string) (*models.Flag, error) {
	return s.GetFlagByKey(context.Background(), projectID, flagKey)
}

func (s *Store) GetFlagByKey(ctx context.Context, projectID, flagKey string) (*models.Flag, error) {
	f := &models.Flag{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, key, name, description, type, default_value,
		        enabled, tags, depends_on, kill_switch, archived, created_by,
		        created_at, updated_at
		 FROM flags WHERE project_id = $1 AND key = $2`,
		projectID, flagKey,
	).Scan(
		&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &f.DefaultValue,
		&f.Enabled, &f.Tags, &f.DependsOn, &f.KillSwitch, &f.Archived, &f.CreatedBy,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err, "flag", projectID+"/"+flagKey)
	}
	return f, nil
}

func (s *Store) GetFlagByID(ctx context.Context, id string) (*models.Flag, error) {
	f := &models.Flag{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, key, name, description, type, default_value,
		        enabled, tags, depends_on, kill_switch, archived, created_by,
		        created_at, updated_at
		 FROM flags WHERE id = $1`, id,
	).Scan(
		&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &f.DefaultValue,
		&f.Enabled, &f.Tags, &f.DependsOn, &f.KillSwitch, &f.Archived, &f.CreatedBy,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err, "flag", id)
	}
	return f, nil
}

// FlagFilter controls which flags are returned by ListFlags.
type FlagFilter struct {
	Tag      string // filter by tag (exact match in the tags array)
	Enabled  *bool  // filter by enabled status
	Archived *bool  // filter by archived status
	Search   string // case-insensitive substring match on key or name
}

func (s *Store) ListFlags(ctx context.Context, projectID string, filter FlagFilter) ([]models.Flag, error) {
	query := strings.Builder{}
	query.WriteString(
		`SELECT id, project_id, key, name, description, type, default_value,
		        enabled, tags, depends_on, kill_switch, archived, created_by,
		        created_at, updated_at
		 FROM flags WHERE project_id = $1`)

	args := []any{projectID}
	argIdx := 2

	if filter.Tag != "" {
		query.WriteString(fmt.Sprintf(` AND $%d = ANY(tags)`, argIdx))
		args = append(args, filter.Tag)
		argIdx++
	}
	if filter.Enabled != nil {
		query.WriteString(fmt.Sprintf(` AND enabled = $%d`, argIdx))
		args = append(args, *filter.Enabled)
		argIdx++
	}
	if filter.Archived != nil {
		query.WriteString(fmt.Sprintf(` AND archived = $%d`, argIdx))
		args = append(args, *filter.Archived)
		argIdx++
	}
	if filter.Search != "" {
		query.WriteString(fmt.Sprintf(` AND (key ILIKE $%d OR name ILIKE $%d)`, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	query.WriteString(` ORDER BY created_at DESC`)

	rows, err := s.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	defer rows.Close()

	var out []models.Flag
	for rows.Next() {
		var f models.Flag
		if err := rows.Scan(
			&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &f.DefaultValue,
			&f.Enabled, &f.Tags, &f.DependsOn, &f.KillSwitch, &f.Archived, &f.CreatedBy,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan flag: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) UpdateFlag(ctx context.Context, f *models.Flag) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE flags SET name = $1, description = $2, type = $3, default_value = $4,
		                  enabled = $5, tags = $6, depends_on = $7, kill_switch = $8,
		                  archived = $9, updated_at = now()
		 WHERE id = $10`,
		f.Name, f.Description, f.Type, f.DefaultValue,
		f.Enabled, f.Tags, f.DependsOn, f.KillSwitch,
		f.Archived, f.ID,
	)
	if err != nil {
		return fmt.Errorf("update flag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "flag", ID: f.ID}
	}
	return nil
}

func (s *Store) DeleteFlag(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM flags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "flag", ID: id}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Flag-Environment configuration
// ---------------------------------------------------------------------------

func (s *Store) CreateFlagEnvironment(ctx context.Context, fe *models.FlagEnvironment) error {
	rulesJSON, err := json.Marshal(fe.Rules)
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}
	fallthroughJSON, err := json.Marshal(fe.Fallthrough)
	if err != nil {
		return fmt.Errorf("marshal fallthrough: %w", err)
	}

	return s.pool.QueryRow(ctx,
		`INSERT INTO flag_environments (flag_id, environment_id, enabled, rules, fallthrough, off_variation)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING version, updated_at`,
		fe.FlagID, fe.EnvironmentID, fe.Enabled, rulesJSON, fallthroughJSON, fe.OffVariation,
	).Scan(&fe.Version, &fe.UpdatedAt)
}

// GetFlagEnvironment retrieves the per-environment config for a flag.
// This satisfies the evaluator FlagStore interface.
func (s *Store) GetFlagEnvironment(flagID, envID string) (*models.FlagEnvironment, error) {
	return s.GetFlagEnvironmentCtx(context.Background(), flagID, envID)
}

func (s *Store) GetFlagEnvironmentCtx(ctx context.Context, flagID, envID string) (*models.FlagEnvironment, error) {
	fe := &models.FlagEnvironment{}
	var rulesJSON, fallthroughJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT flag_id, environment_id, enabled, rules, fallthrough, off_variation, version, updated_at
		 FROM flag_environments WHERE flag_id = $1 AND environment_id = $2`,
		flagID, envID,
	).Scan(
		&fe.FlagID, &fe.EnvironmentID, &fe.Enabled,
		&rulesJSON, &fallthroughJSON, &fe.OffVariation,
		&fe.Version, &fe.UpdatedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err, "flag_environment", flagID+"/"+envID)
	}

	if err := json.Unmarshal(rulesJSON, &fe.Rules); err != nil {
		return nil, fmt.Errorf("unmarshal rules: %w", err)
	}
	if err := json.Unmarshal(fallthroughJSON, &fe.Fallthrough); err != nil {
		return nil, fmt.Errorf("unmarshal fallthrough: %w", err)
	}
	return fe, nil
}

// UpdateFlagEnvironment updates the per-env config and increments the version.
func (s *Store) UpdateFlagEnvironment(ctx context.Context, fe *models.FlagEnvironment) error {
	rulesJSON, err := json.Marshal(fe.Rules)
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}
	fallthroughJSON, err := json.Marshal(fe.Fallthrough)
	if err != nil {
		return fmt.Errorf("marshal fallthrough: %w", err)
	}

	err = s.pool.QueryRow(ctx,
		`UPDATE flag_environments
		 SET enabled = $1, rules = $2, fallthrough = $3, off_variation = $4,
		     version = version + 1, updated_at = now()
		 WHERE flag_id = $5 AND environment_id = $6
		 RETURNING version, updated_at`,
		fe.Enabled, rulesJSON, fallthroughJSON, fe.OffVariation,
		fe.FlagID, fe.EnvironmentID,
	).Scan(&fe.Version, &fe.UpdatedAt)
	if err != nil {
		return wrapNotFound(err, "flag_environment", fe.FlagID+"/"+fe.EnvironmentID)
	}
	return nil
}

func (s *Store) DeleteFlagEnvironment(ctx context.Context, flagID, envID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM flag_environments WHERE flag_id = $1 AND environment_id = $2`,
		flagID, envID,
	)
	if err != nil {
		return fmt.Errorf("delete flag_environment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "flag_environment", ID: flagID + "/" + envID}
	}
	return nil
}

// GetAllFlagsForEnvironment returns all non-archived flags for a project along
// with their per-environment configs. Satisfies the evaluator FlagStore interface.
func (s *Store) GetAllFlagsForEnvironment(projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error) {
	return s.GetAllFlagsForEnvironmentCtx(context.Background(), projectID, envID)
}

func (s *Store) GetAllFlagsForEnvironmentCtx(ctx context.Context, projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error) {
	// Fetch flags.
	rows, err := s.pool.Query(ctx,
		`SELECT f.id, f.project_id, f.key, f.name, f.description, f.type, f.default_value,
		        f.enabled, f.tags, f.depends_on, f.kill_switch, f.archived, f.created_by,
		        f.created_at, f.updated_at
		 FROM flags f
		 WHERE f.project_id = $1 AND f.archived = false
		 ORDER BY f.key`, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("get all flags: %w", err)
	}
	defer rows.Close()

	var flags []models.Flag
	for rows.Next() {
		var f models.Flag
		if err := rows.Scan(
			&f.ID, &f.ProjectID, &f.Key, &f.Name, &f.Description, &f.Type, &f.DefaultValue,
			&f.Enabled, &f.Tags, &f.DependsOn, &f.KillSwitch, &f.Archived, &f.CreatedBy,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan flag: %w", err)
		}
		flags = append(flags, f)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Fetch flag_environments for the given environment.
	feRows, err := s.pool.Query(ctx,
		`SELECT fe.flag_id, fe.environment_id, fe.enabled, fe.rules, fe.fallthrough,
		        fe.off_variation, fe.version, fe.updated_at
		 FROM flag_environments fe
		 JOIN flags f ON f.id = fe.flag_id
		 WHERE f.project_id = $1 AND fe.environment_id = $2 AND f.archived = false`,
		projectID, envID)
	if err != nil {
		return nil, nil, fmt.Errorf("get all flag_environments: %w", err)
	}
	defer feRows.Close()

	var flagEnvs []models.FlagEnvironment
	for feRows.Next() {
		var fe models.FlagEnvironment
		var rulesJSON, fallthroughJSON []byte
		if err := feRows.Scan(
			&fe.FlagID, &fe.EnvironmentID, &fe.Enabled,
			&rulesJSON, &fallthroughJSON, &fe.OffVariation,
			&fe.Version, &fe.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan flag_environment: %w", err)
		}
		if err := json.Unmarshal(rulesJSON, &fe.Rules); err != nil {
			return nil, nil, fmt.Errorf("unmarshal rules: %w", err)
		}
		if err := json.Unmarshal(fallthroughJSON, &fe.Fallthrough); err != nil {
			return nil, nil, fmt.Errorf("unmarshal fallthrough: %w", err)
		}
		flagEnvs = append(flagEnvs, fe)
	}
	return flags, flagEnvs, feRows.Err()
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func (s *Store) CreateUser(ctx context.Context, u *models.User) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, role, api_key)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, created_at`,
		u.Email, u.Name, u.Role, u.APIKey,
	).Scan(&u.ID, &u.CreatedAt)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.GetUser(ctx, id)
}

func (s *Store) GetUser(ctx context.Context, id string) (*models.User, error) {
	u := &models.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, api_key, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.APIKey, &u.CreatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "user", id)
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, api_key, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.APIKey, &u.CreatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "user", email)
	}
	return u, nil
}

func (s *Store) GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	u := &models.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, api_key, created_at FROM users WHERE api_key = $1`, apiKey,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.APIKey, &u.CreatedAt)
	if err != nil {
		return nil, wrapNotFound(err, "user", "api_key")
	}
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, name, role, api_key, created_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.APIKey, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UpdateUser(ctx context.Context, u *models.User) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET email = $1, name = $2, role = $3 WHERE id = $4`,
		u.Email, u.Name, u.Role, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "user", ID: u.ID}
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "user", ID: id}
	}
	return nil
}

// ---------------------------------------------------------------------------
// User-Project membership
// ---------------------------------------------------------------------------

func (s *Store) AddUserToProject(ctx context.Context, userID, projectID string, role models.Role) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_projects (user_id, project_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, project_id) DO UPDATE SET role = EXCLUDED.role`,
		userID, projectID, role,
	)
	if err != nil {
		return fmt.Errorf("add user to project: %w", err)
	}
	return nil
}

func (s *Store) RemoveUserFromProject(ctx context.Context, userID, projectID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM user_projects WHERE user_id = $1 AND project_id = $2`,
		userID, projectID,
	)
	if err != nil {
		return fmt.Errorf("remove user from project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "user_project", ID: userID + "/" + projectID}
	}
	return nil
}

func (s *Store) GetUserProjectRole(ctx context.Context, userID, projectID string) (models.Role, error) {
	var role models.Role
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM user_projects WHERE user_id = $1 AND project_id = $2`,
		userID, projectID,
	).Scan(&role)
	if err != nil {
		return "", wrapNotFound(err, "user_project", userID+"/"+projectID)
	}
	return role, nil
}

// ---------------------------------------------------------------------------
// Experiments
// ---------------------------------------------------------------------------

func (s *Store) CreateExperiment(ctx context.Context, exp *models.Experiment) error {
	metricsJSON, err := json.Marshal(exp.Metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	guardrailJSON, err := json.Marshal(exp.GuardrailMetrics)
	if err != nil {
		return fmt.Errorf("marshal guardrail metrics: %w", err)
	}

	return s.pool.QueryRow(ctx,
		`INSERT INTO experiments
		 (project_id, flag_id, environment_id, name, description, status,
		  hypothesis, traffic_percent, metrics, guardrail_metrics)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, created_at, updated_at`,
		exp.ProjectID, exp.FlagID, exp.EnvironmentID, exp.Name, exp.Description,
		exp.Status, exp.Hypothesis, exp.TrafficPercent, metricsJSON, guardrailJSON,
	).Scan(&exp.ID, &exp.CreatedAt, &exp.UpdatedAt)
}

func (s *Store) GetExperiment(ctx context.Context, id string) (*models.Experiment, error) {
	exp := &models.Experiment{}
	var metricsJSON, guardrailJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, flag_id, environment_id, name, description, status,
		        hypothesis, traffic_percent, metrics, guardrail_metrics,
		        started_at, stopped_at, created_at, updated_at
		 FROM experiments WHERE id = $1`, id,
	).Scan(
		&exp.ID, &exp.ProjectID, &exp.FlagID, &exp.EnvironmentID,
		&exp.Name, &exp.Description, &exp.Status, &exp.Hypothesis, &exp.TrafficPercent,
		&metricsJSON, &guardrailJSON,
		&exp.StartedAt, &exp.StoppedAt, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if err != nil {
		return nil, wrapNotFound(err, "experiment", id)
	}
	if err := json.Unmarshal(metricsJSON, &exp.Metrics); err != nil {
		return nil, fmt.Errorf("unmarshal metrics: %w", err)
	}
	if err := json.Unmarshal(guardrailJSON, &exp.GuardrailMetrics); err != nil {
		return nil, fmt.Errorf("unmarshal guardrail metrics: %w", err)
	}
	return exp, nil
}

func (s *Store) ListExperiments(ctx context.Context, projectID string) ([]models.Experiment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, project_id, flag_id, environment_id, name, description, status,
		        hypothesis, traffic_percent, metrics, guardrail_metrics,
		        started_at, stopped_at, created_at, updated_at
		 FROM experiments WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()

	var out []models.Experiment
	for rows.Next() {
		var exp models.Experiment
		var metricsJSON, guardrailJSON []byte
		if err := rows.Scan(
			&exp.ID, &exp.ProjectID, &exp.FlagID, &exp.EnvironmentID,
			&exp.Name, &exp.Description, &exp.Status, &exp.Hypothesis, &exp.TrafficPercent,
			&metricsJSON, &guardrailJSON,
			&exp.StartedAt, &exp.StoppedAt, &exp.CreatedAt, &exp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		if err := json.Unmarshal(metricsJSON, &exp.Metrics); err != nil {
			return nil, fmt.Errorf("unmarshal metrics: %w", err)
		}
		if err := json.Unmarshal(guardrailJSON, &exp.GuardrailMetrics); err != nil {
			return nil, fmt.Errorf("unmarshal guardrail metrics: %w", err)
		}
		out = append(out, exp)
	}
	return out, rows.Err()
}

func (s *Store) UpdateExperiment(ctx context.Context, exp *models.Experiment) error {
	metricsJSON, err := json.Marshal(exp.Metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	guardrailJSON, err := json.Marshal(exp.GuardrailMetrics)
	if err != nil {
		return fmt.Errorf("marshal guardrail metrics: %w", err)
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE experiments
		 SET name = $1, description = $2, status = $3, hypothesis = $4,
		     traffic_percent = $5, metrics = $6, guardrail_metrics = $7,
		     started_at = $8, stopped_at = $9, updated_at = now()
		 WHERE id = $10`,
		exp.Name, exp.Description, exp.Status, exp.Hypothesis,
		exp.TrafficPercent, metricsJSON, guardrailJSON,
		exp.StartedAt, exp.StoppedAt, exp.ID,
	)
	if err != nil {
		return fmt.Errorf("update experiment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "experiment", ID: exp.ID}
	}
	return nil
}

func (s *Store) DeleteExperiment(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM experiments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete experiment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return &NotFoundError{Resource: "experiment", ID: id}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Audit Logs
// ---------------------------------------------------------------------------

func (s *Store) WriteAuditLog(ctx context.Context, entry *models.AuditEntry) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO audit_logs
		 (project_id, environment_id, action, actor_id, actor_email,
		  resource_type, resource_id, previous_state, new_state)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, timestamp`,
		entry.ProjectID, nilIfEmpty(entry.EnvironmentID), entry.Action, entry.ActorID,
		entry.ActorEmail, entry.ResourceType, entry.ResourceID,
		entry.PreviousState, entry.NewState,
	).Scan(&entry.ID, &entry.Timestamp)
}

// AuditLogQuery controls pagination and filtering for audit log queries.
type AuditLogQuery struct {
	ProjectID    string
	ResourceType string // optional filter
	ResourceID   string // optional filter
	Limit        int
	Offset       int
}

func (s *Store) QueryAuditLogs(ctx context.Context, q AuditLogQuery) ([]models.AuditEntry, int64, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 500 {
		q.Limit = 500
	}

	where := strings.Builder{}
	where.WriteString(`WHERE project_id = $1`)
	args := []any{q.ProjectID}
	argIdx := 2

	if q.ResourceType != "" {
		where.WriteString(fmt.Sprintf(` AND resource_type = $%d`, argIdx))
		args = append(args, q.ResourceType)
		argIdx++
	}
	if q.ResourceID != "" {
		where.WriteString(fmt.Sprintf(` AND resource_id = $%d`, argIdx))
		args = append(args, q.ResourceID)
		argIdx++
	}

	// Total count.
	var total int64
	countSQL := `SELECT count(*) FROM audit_logs ` + where.String()
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	// Fetch page.
	dataSQL := fmt.Sprintf(
		`SELECT id, project_id, environment_id, action, actor_id, actor_email,
		        resource_type, resource_id, previous_state, new_state, timestamp
		 FROM audit_logs %s
		 ORDER BY timestamp DESC
		 LIMIT $%d OFFSET $%d`,
		where.String(), argIdx, argIdx+1,
	)
	args = append(args, q.Limit, q.Offset)

	rows, err := s.pool.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var out []models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		var envID *string
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &envID, &e.Action, &e.ActorID, &e.ActorEmail,
			&e.ResourceType, &e.ResourceID, &e.PreviousState, &e.NewState, &e.Timestamp,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit entry: %w", err)
		}
		if envID != nil {
			e.EnvironmentID = *envID
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

// NotFoundError is returned when a resource does not exist.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// IsNotFound reports whether err is a NotFoundError.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotFoundError)
	return ok
}

func wrapNotFound(err error, resource, id string) error {
	if err == pgx.ErrNoRows {
		return &NotFoundError{Resource: resource, ID: id}
	}
	return fmt.Errorf("get %s: %w", resource, err)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ensure Store satisfies the evaluator's FlagStore interface at compile time.
// This uses a blank import-free approach: the interface is checked where it's used,
// but we add a compile-time assertion here for safety.
var _ interface {
	GetFlag(projectID, flagKey string) (*models.Flag, error)
	GetFlagEnvironment(flagID, envID string) (*models.FlagEnvironment, error)
	GetAllFlagsForEnvironment(projectID, envID string) ([]models.Flag, []models.FlagEnvironment, error)
} = (*Store)(nil)

