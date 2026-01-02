package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vsamtuc/mcm/pkg/resource"
)

type ResourceStore struct {
	db *sql.DB
}

func NewResourceStore(db *sql.DB) *ResourceStore {
	return &ResourceStore{db: db}
}

func (s *ResourceStore) CreateResourceClass(ctx context.Context, rc resource.ResourceClass) (resource.ResourceClass, error) {
	err := s.db.QueryRowContext(ctx, `
        INSERT INTO resource_classes (name, version, description)
        VALUES ($1, $2, $3)
        RETURNING id`,
		rc.Name, rc.Version, rc.Description,
	).Scan(&rc.ID)
	return rc, err
}

func (s *ResourceStore) GetResourceClass(ctx context.Context, id int64) (resource.ResourceClass, error) {
	var rc resource.ResourceClass
	err := s.db.QueryRowContext(ctx, `
        SELECT id, name, version, description
        FROM resource_classes
        WHERE id = $1`, id,
	).Scan(&rc.ID, &rc.Name, &rc.Version, &rc.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ResourceClass{}, resource.ErrNotFound
	}
	return rc, err
}

func (s *ResourceStore) ListResourceClasses(ctx context.Context) ([]resource.ResourceClass, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, name, version, description
        FROM resource_classes
        ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []resource.ResourceClass
	for rows.Next() {
		var rc resource.ResourceClass
		if err := rows.Scan(&rc.ID, &rc.Name, &rc.Version, &rc.Description); err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	return out, rows.Err()
}

func (s *ResourceStore) UpdateResourceClass(ctx context.Context, rc resource.ResourceClass) error {
	res, err := s.db.ExecContext(ctx, `
        UPDATE resource_classes
        SET name = $1, version = $2, description = $3
        WHERE id = $4`,
		rc.Name, rc.Version, rc.Description, rc.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return resource.ErrNotFound
	}
	return nil
}

func (s *ResourceStore) DeleteResourceClass(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
        DELETE FROM resource_classes
        WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return resource.ErrNotFound
	}
	return nil
}

func (s *ResourceStore) CreateResourceSet(ctx context.Context, rs resource.ResourceSet) (resource.ResourceSet, error) {
	if rs.Spec == nil {
		rs.Spec = []byte("{}")
	}
	if rs.Status == nil {
		rs.Status = []byte("{}")
	}

	err := s.db.QueryRowContext(ctx, `
        INSERT INTO resource_sets (course_id, resource_class_id, owner_type, spec, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, last_updated`,
		rs.CourseID, rs.ResourceClassID, rs.OwnerType, rs.Spec, rs.Status,
	).Scan(&rs.ID, &rs.CreatedAt, &rs.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ResourceSet{}, resource.ErrNotFound
	}
	return rs, err
}

func (s *ResourceStore) GetResourceSet(ctx context.Context, id int64) (resource.ResourceSet, error) {
	var rs resource.ResourceSet
	err := s.db.QueryRowContext(ctx, `
        SELECT id, course_id, resource_class_id, owner_type, spec, status, created_at, last_updated
        FROM resource_sets
        WHERE id = $1`, id,
	).Scan(&rs.ID, &rs.CourseID, &rs.ResourceClassID, &rs.OwnerType, &rs.Spec, &rs.Status, &rs.CreatedAt, &rs.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ResourceSet{}, resource.ErrNotFound
	}
	return rs, err
}

func (s *ResourceStore) ListResourceSetsByCourse(ctx context.Context, courseID int64) ([]resource.ResourceSet, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, course_id, resource_class_id, owner_type, spec, status, created_at, last_updated
        FROM resource_sets
        WHERE course_id = $1
        ORDER BY id`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []resource.ResourceSet
	for rows.Next() {
		var rs resource.ResourceSet
		if err := rows.Scan(&rs.ID, &rs.CourseID, &rs.ResourceClassID, &rs.OwnerType, &rs.Spec, &rs.Status, &rs.CreatedAt, &rs.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, rows.Err()
}

func (s *ResourceStore) UpdateResourceSet(ctx context.Context, rs resource.ResourceSet) error {
	if rs.Spec == nil {
		rs.Spec = []byte("{}")
	}
	if rs.Status == nil {
		rs.Status = []byte("{}")
	}

	err := s.db.QueryRowContext(ctx, `
        UPDATE resource_sets
        SET course_id = $1,
            resource_class_id = $2,
            owner_type = $3,
            spec = $4,
            status = $5,
            last_updated = NOW()
        WHERE id = $6
        RETURNING last_updated`,
		rs.CourseID, rs.ResourceClassID, rs.OwnerType, rs.Spec, rs.Status, rs.ID,
	).Scan(&rs.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ErrNotFound
	}
	return err
}

func (s *ResourceStore) DeleteResourceSet(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
        DELETE FROM resource_sets
        WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return resource.ErrNotFound
	}
	return nil
}

func (s *ResourceStore) CreateResource(ctx context.Context, r resource.Resource) (resource.Resource, error) {
	if r.Spec == nil {
		r.Spec = []byte("{}")
	}
	if r.Status == nil {
		r.Status = []byte("{}")
	}

	err := s.db.QueryRowContext(ctx, `
        INSERT INTO resources (resource_set_id, resource_class_id, spec, status)
        SELECT $1, rs.resource_class_id, $2, $3
        FROM resource_sets rs
        WHERE rs.id = $1
        RETURNING id, resource_class_id, created_at, last_updated`,
		r.ResourceSetID, r.Spec, r.Status,
	).Scan(&r.ID, &r.ResourceClassID, &r.CreatedAt, &r.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	}
	return r, err
}

func (s *ResourceStore) GetResource(ctx context.Context, id int64) (resource.Resource, error) {
	var r resource.Resource
	err := s.db.QueryRowContext(ctx, `
        SELECT id, resource_set_id, resource_class_id, spec, status, created_at, last_updated
        FROM resources
        WHERE id = $1`, id,
	).Scan(&r.ID, &r.ResourceSetID, &r.ResourceClassID, &r.Spec, &r.Status, &r.CreatedAt, &r.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.Resource{}, resource.ErrNotFound
	}
	return r, err
}

func (s *ResourceStore) ListResourcesBySet(ctx context.Context, resourceSetID int64) ([]resource.Resource, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, resource_set_id, resource_class_id, spec, status, created_at, last_updated
        FROM resources
        WHERE resource_set_id = $1
        ORDER BY id`, resourceSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []resource.Resource
	for rows.Next() {
		var r resource.Resource
		if err := rows.Scan(&r.ID, &r.ResourceSetID, &r.ResourceClassID, &r.Spec, &r.Status, &r.CreatedAt, &r.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *ResourceStore) UpdateResource(ctx context.Context, r resource.Resource) error {
	if r.Spec == nil {
		r.Spec = []byte("{}")
	}
	if r.Status == nil {
		r.Status = []byte("{}")
	}

	err := s.db.QueryRowContext(ctx, `
        UPDATE resources
        SET spec = $1,
            status = $2,
            last_updated = NOW()
        WHERE id = $3
        RETURNING resource_set_id, resource_class_id, created_at, last_updated`,
		r.Spec, r.Status, r.ID,
	).Scan(&r.ResourceSetID, &r.ResourceClassID, &r.CreatedAt, &r.LastUpdated)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ErrNotFound
	}
	return err
}

func (s *ResourceStore) DeleteResource(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
        DELETE FROM resources
        WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return resource.ErrNotFound
	}
	return nil
}
