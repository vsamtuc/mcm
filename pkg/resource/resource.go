package resource

import (
	"encoding/json"
	"time"
)

/*
	The  mcm application supports resource management for students and teams.
	There are three entities involved:
	ResourceClass, ResourceSet and Resource.

	A ResourceClass defines the type of resource, e.g., "K8s namespace Instance",
	"Storage Bucket", "DNS name", etc.

	Each ResourceSet is associated with a ResourceClass, and defines a set of resources
	of that class. For example, a ResourceSet of class "K8s namespace Instance" might
	define a set of namespaces for a particular course.
	Each ResourceSet is associated with a single Course. A Course may have multiple
	ResourceSets of different classes, or even multiple ResourceSets of the same class.

	At the storage layer, we store the ResourceSet metadata, which consists of:
	- ResourceSet ID
	- Course ID
	- ResourceClass ID
	- A "spec" JSON object which contains configuration parameters for the ResourceSet
	(e.g., size, region, etc.)
	- A "status" JSON object which contains the current status of the ResourceSet
	(e.g., provisioning status, error messages, etc.)

	A Resource represents a single instance of a resource in a ResourceSet. For example,
	a Resource of class "K8s namespace Instance" might represent a single namespace
	created for a student or team.

	Each resource is associated with either:
	- the Course itself,
	- a Student,
	- a Team,
	- a Student as member of a team.

	At the storage layer, we store the resource metadata, which consists of:
	- Resource ID
	- ResourceSet ID
	- ResourceClass ID
	- Owner Type (Student, Team, StudentInTeam)
	- The "spec" of the resource, which is a JSON blob that contains the details
	  of the resource (e.g., namespace name, bucket name, etc.)
	- The "status" of the resource, which is a JSON blob that contains the current
	  status of the resource (e.g., provisioned, failed, etc.)

	The actual provisioning and management of the resources is handled by
	dedicated operators for each ResourceClass. The details of these operators
	are outside the scope of this code.

	TODO: Provide pointers to the operator spec.
*/

type ResourceClass struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type ResourceOwnerType string

const (
	OwnerTypeCourse        ResourceOwnerType = "Course"
	OwnerTypeStudent       ResourceOwnerType = "Student"
	OwnerTypeTeam          ResourceOwnerType = "Team"
	OwnerTypeStudentInTeam ResourceOwnerType = "StudentInTeam"
)

type ResourceSet struct {
	ID              int64              `json:"id"`
	CourseID        int64              `json:"course_id"`
	ResourceClassID int64              `json:"resource_class_id"`
	OwnerType       ResourceOwnerType  `json:"owner_type"`
	Spec            json.RawMessage    `json:"spec"`
	Status          json.RawMessage    `json:"status"`

	// some extra metadata
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
}

type Resource struct {
	ID              int64           `json:"id"`
	ResourceSetID   int64           `json:"resource_set_id"`
	ResourceClassID int64           `json:"resource_class_id"`
	Spec            json.RawMessage `json:"spec"`
	Status          json.RawMessage `json:"status"`

	// some extra metadata
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
}
