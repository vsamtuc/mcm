package memory

import (
	"context"
	"sync"
	"time"

	"github.com/vsamtuc/mcm/pkg/resource"
)

type ResourceStore struct {
	mu             sync.RWMutex
	nextClassID    int64
	nextSetID      int64
	nextResourceID int64
	classes        map[int64]resource.ResourceClass
	sets           map[int64]resource.ResourceSet
	resources      map[int64]resource.Resource
}

func NewResourceStore() *ResourceStore {
	return &ResourceStore{
		nextClassID:    1,
		nextSetID:      1,
		nextResourceID: 1,
		classes:        make(map[int64]resource.ResourceClass),
		sets:           make(map[int64]resource.ResourceSet),
		resources:      make(map[int64]resource.Resource),
	}
}

func (s *ResourceStore) CreateResourceClass(ctx context.Context, rc resource.ResourceClass) (resource.ResourceClass, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rc.ID = s.nextClassID
	s.nextClassID++
	s.classes[rc.ID] = rc
	return rc, nil
}

func (s *ResourceStore) GetResourceClass(ctx context.Context, id int64) (resource.ResourceClass, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rc, ok := s.classes[id]
	if !ok {
		return resource.ResourceClass{}, resource.ErrNotFound
	}
	return rc, nil
}

func (s *ResourceStore) ListResourceClasses(ctx context.Context) ([]resource.ResourceClass, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]resource.ResourceClass, 0, len(s.classes))
	for _, rc := range s.classes {
		out = append(out, rc)
	}
	return out, nil
}

func (s *ResourceStore) UpdateResourceClass(ctx context.Context, rc resource.ResourceClass) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.classes[rc.ID]; !ok {
		return resource.ErrNotFound
	}
	s.classes[rc.ID] = rc
	return nil
}

func (s *ResourceStore) DeleteResourceClass(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.classes[id]; !ok {
		return resource.ErrNotFound
	}
	delete(s.classes, id)

	for setID, rs := range s.sets {
		if rs.ResourceClassID == id {
			delete(s.sets, setID)
			for resID, r := range s.resources {
				if r.ResourceSetID == setID {
					delete(s.resources, resID)
				}
			}
		}
	}

	return nil
}

func (s *ResourceStore) CreateResourceSet(ctx context.Context, rs resource.ResourceSet) (resource.ResourceSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.classes[rs.ResourceClassID]; !ok {
		return resource.ResourceSet{}, resource.ErrNotFound
	}

	now := time.Now().UTC()
	rs.ID = s.nextSetID
	s.nextSetID++
	rs.CreatedAt = now
	rs.LastUpdated = now
	if rs.Spec == nil {
		rs.Spec = []byte("{}")
	}
	if rs.Status == nil {
		rs.Status = []byte("{}")
	}
	s.sets[rs.ID] = rs
	return rs, nil
}

func (s *ResourceStore) GetResourceSet(ctx context.Context, id int64) (resource.ResourceSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rs, ok := s.sets[id]
	if !ok {
		return resource.ResourceSet{}, resource.ErrNotFound
	}
	return rs, nil
}

func (s *ResourceStore) ListResourceSetsByCourse(ctx context.Context, courseID int64) ([]resource.ResourceSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]resource.ResourceSet, 0)
	for _, rs := range s.sets {
		if rs.CourseID == courseID {
			out = append(out, rs)
		}
	}
	return out, nil
}

func (s *ResourceStore) UpdateResourceSet(ctx context.Context, rs resource.ResourceSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sets[rs.ID]; !ok {
		return resource.ErrNotFound
	}
	if rs.Spec == nil {
		rs.Spec = []byte("{}")
	}
	if rs.Status == nil {
		rs.Status = []byte("{}")
	}
	rs.LastUpdated = time.Now().UTC()
	s.sets[rs.ID] = rs
	return nil
}

func (s *ResourceStore) DeleteResourceSet(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sets[id]; !ok {
		return resource.ErrNotFound
	}
	delete(s.sets, id)
	for resID, r := range s.resources {
		if r.ResourceSetID == id {
			delete(s.resources, resID)
		}
	}
	return nil
}

func (s *ResourceStore) CreateResource(ctx context.Context, r resource.Resource) (resource.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rs, ok := s.sets[r.ResourceSetID]
	if !ok {
		return resource.Resource{}, resource.ErrNotFound
	}

	now := time.Now().UTC()
	r.ID = s.nextResourceID
	s.nextResourceID++
	r.ResourceClassID = rs.ResourceClassID
	r.CreatedAt = now
	r.LastUpdated = now
	if r.Spec == nil {
		r.Spec = []byte("{}")
	}
	if r.Status == nil {
		r.Status = []byte("{}")
	}
	s.resources[r.ID] = r
	return r, nil
}

func (s *ResourceStore) GetResource(ctx context.Context, id int64) (resource.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.resources[id]
	if !ok {
		return resource.Resource{}, resource.ErrNotFound
	}
	return r, nil
}

func (s *ResourceStore) ListResourcesBySet(ctx context.Context, resourceSetID int64) ([]resource.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]resource.Resource, 0)
	for _, r := range s.resources {
		if r.ResourceSetID == resourceSetID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *ResourceStore) UpdateResource(ctx context.Context, r resource.Resource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.resources[r.ID]
	if !ok {
		return resource.ErrNotFound
	}
	if r.Spec == nil {
		r.Spec = []byte("{}")
	}
	if r.Status == nil {
		r.Status = []byte("{}")
	}
	r.ResourceSetID = existing.ResourceSetID
	r.ResourceClassID = existing.ResourceClassID
	r.CreatedAt = existing.CreatedAt
	r.LastUpdated = time.Now().UTC()
	s.resources[r.ID] = r
	return nil
}

func (s *ResourceStore) DeleteResource(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.resources[id]; !ok {
		return resource.ErrNotFound
	}
	delete(s.resources, id)
	return nil
}
