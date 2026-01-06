package application

import (
	"context"

	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/resource"
)

// Service exposes the application use cases to the transport layer.
type Service interface {

	// Manage users
	ListProfessors(ctx context.Context) ([]course.Professor, error)
	ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error)

	// Course management
	ListCourses(ctx context.Context) ([]course.Course, error)
	GetCourse(ctx context.Context, id int64) (course.Course, error)
	SetCourseActive(ctx context.Context, id int64) (course.Course, error)
	SetCourseInactive(ctx context.Context, id int64) (course.Course, error)
	CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error)
	UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error)
	DeleteCourse(ctx context.Context, id int64) error

	// Course instructor management
	AddCourseInstructor(ctx context.Context, courseID int64, instructor course.Instructor) (course.Course, error)
	RemoveCourseInstructor(ctx context.Context, courseID int64, professorID int64) (course.Course, error)

	// Students
	ListStudents(ctx context.Context) ([]course.Student, error)
	StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error)
	EnrollStudentInCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error)
	UnenrollStudentFromCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error)

	// Teams
	ListTeams(ctx context.Context, courseID int64) ([]course.Team, error)
	CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error)
	UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error)
	DeleteTeam(ctx context.Context, teamID int64) error

	TeamAddMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)
	TeamRemoveMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)

	/*
	   Resource management, generic API.

	   This part of the API manages resource sets and resources in a resource-agnostic way.
	   The actual resource provisioning and deprovisioning is handled by dedicated
	   resource operators, which are outside the scope of this code.
	*/

	// Resource allocation and release. The AllocateResource() call allocates a resource
	// from the given ResourceSet, with the given spec. The owner of the resource is extracted
	// from the caller, which must be a student. If the resource set allocates team resources,
	// the resource is allocated for the team the student belongs to.
	AllocateResource(ctx context.Context,
		resourceSetID int64,
		spec map[string]interface{}) (resource.Resource, error)

	// This call allocates a resource for a specific owner (student, team or team member).
	// The ownerID field is interpreted based on the ResourceSet's OwnerType.
	// This is useful for instructors or admins who want to allocate resources on behalf of others.
	AllocateResourceForOwner(ctx context.Context,
		resourceSetID int64,
		ownerID int64,
		spec map[string]interface{}) (resource.Resource, error)

	ReleaseResource(ctx context.Context, resourceID int64) error

	// Get a resource by ID. The resource can be owned by a student, team, or team member.
	// The caller must have access to the resource, either as a student owning it, as a member
	// of the team owning it, or as an instructor of the course.
	GetResource(ctx context.Context, id int64) (resource.Resource, error)

	// List the resources owned by a particular student. All resources in all courses will
	// be returned.
	ListResourcesByStudent(ctx context.Context, studentID int64) ([]resource.Resource, error)

	// List the resources owned by a particular team. Caller must be an instructor of the
	// course the team belongs to, or a student member of the team.
	ListResourcesByTeam(ctx context.Context, teamID int64) ([]resource.Resource, error)

	// List the resources associated with a particular course and the caller,
	// which must be a student enrolled in the course.
	// This call returns both owned resources and resources allocated to team
	// the student belongs to.
	ListCourseResources(ctx context.Context, courseID int64) ([]resource.Resource, error)

	// List the resources associated with a particular resource set. Caller must be an
	// instructor of the course the resource set belongs to.
	ListResourcesBySet(ctx context.Context, resourceSetID int64) ([]resource.Resource, error)

	/*
		Resource set management
	*/
	// Create a ResourceSet for the given course and resource class.
	CreateResourceSet(ctx context.Context,
		courseID int64, rcID int64, ownerType resource.ResourceOwnerType,
		spec map[string]interface{}) (resource.ResourceSet, error)

	// Return all resource sets associated with a given course. Caller must be
	// an instructor of the course, or a student enrolled in the course (or an admin).
	ListResourceSetsByCourse(ctx context.Context, courseID int64) ([]resource.ResourceSet, error)

	// Return all resource sets associated with a given resource class. Caller must be an
	// admin.
	ListResourceSetsByClass(ctx context.Context, resourceClassID int64) ([]resource.ResourceSet, error)

	/*
		Resource class lifecycle management

		Resource classes are defined by the corresponding resource driver operations
		during initialization. Therefore, we only expose listing as an API operation.
	*/
	ListResourceClasses(ctx context.Context) ([]resource.ResourceClass, error)
}
