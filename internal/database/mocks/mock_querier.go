package mocks

import (
	"context"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
)

type MockQuerier struct {
	GetPostFunc                func(ctx context.Context, slug string) (database.Post, error)
	InsertPostFunc             func(ctx context.Context, arg database.InsertPostParams) (database.Post, error)
	ListPostsFunc              func(ctx context.Context) ([]database.Post, error)
	ListPostsFilteredFunc      func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error)
	ListDistinctTagsFunc       func(ctx context.Context) ([]string, error)
	UpdatePostFunc             func(ctx context.Context, arg database.UpdatePostParams) (database.Post, error)
	DeletePostFunc             func(ctx context.Context, id int64) (int64, error)
	InsertProjectFunc          func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error)
	GetProjectBySlugFunc       func(ctx context.Context, slug string) (database.Project, error)
	ListProjectsFunc           func(ctx context.Context) ([]database.Project, error)
	ListProjectsFilteredFunc   func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error)
	GetProjectsByIDsFunc       func(ctx context.Context, ids []int64) ([]database.Project, error)
	GetProjectsForPostFunc     func(ctx context.Context, postID int64) ([]database.Project, error)
	ListPostsByProjectSlugFunc func(ctx context.Context, slug string) ([]database.Post, error)
	DeletePostProjectsFunc     func(ctx context.Context, postID int64) error
	InsertPostProjectFunc      func(ctx context.Context, arg database.InsertPostProjectParams) error
	ListFeaturedPostsFunc      func(ctx context.Context) ([]database.Post, error)
	ClearFeaturedPostsFunc     func(ctx context.Context) error
	SetFeaturedPostFunc        func(ctx context.Context, arg database.SetFeaturedPostParams) error
	ListFeaturedProjectsFunc   func(ctx context.Context) ([]database.Project, error)
	ClearFeaturedProjectsFunc  func(ctx context.Context) error
	SetFeaturedProjectFunc     func(ctx context.Context, arg database.SetFeaturedProjectParams) error
}

func (m *MockQuerier) GetPost(ctx context.Context, slug string) (database.Post, error) {
	return m.GetPostFunc(ctx, slug)
}

func (m *MockQuerier) InsertPost(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
	return m.InsertPostFunc(ctx, arg)
}

func (m *MockQuerier) ListPosts(ctx context.Context) ([]database.Post, error) {
	return m.ListPostsFunc(ctx)
}

func (m *MockQuerier) ListPostsFiltered(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
	return m.ListPostsFilteredFunc(ctx, arg)
}

func (m *MockQuerier) ListDistinctTags(ctx context.Context) ([]string, error) {
	return m.ListDistinctTagsFunc(ctx)
}

func (m *MockQuerier) UpdatePost(ctx context.Context, arg database.UpdatePostParams) (database.Post, error) {
	return m.UpdatePostFunc(ctx, arg)
}

func (m *MockQuerier) DeletePost(ctx context.Context, id int64) (int64, error) {
	return m.DeletePostFunc(ctx, id)
}

func (m *MockQuerier) InsertProject(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
	return m.InsertProjectFunc(ctx, arg)
}

func (m *MockQuerier) GetProjectBySlug(ctx context.Context, slug string) (database.Project, error) {
	return m.GetProjectBySlugFunc(ctx, slug)
}

func (m *MockQuerier) ListProjects(ctx context.Context) ([]database.Project, error) {
	return m.ListProjectsFunc(ctx)
}

func (m *MockQuerier) ListProjectsFiltered(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
	return m.ListProjectsFilteredFunc(ctx, arg)
}

func (m *MockQuerier) GetProjectsByIDs(ctx context.Context, ids []int64) ([]database.Project, error) {
	return m.GetProjectsByIDsFunc(ctx, ids)
}

func (m *MockQuerier) GetProjectsForPost(ctx context.Context, postID int64) ([]database.Project, error) {
	return m.GetProjectsForPostFunc(ctx, postID)
}

func (m *MockQuerier) ListPostsByProjectSlug(ctx context.Context, slug string) ([]database.Post, error) {
	return m.ListPostsByProjectSlugFunc(ctx, slug)
}

func (m *MockQuerier) DeletePostProjects(ctx context.Context, postID int64) error {
	return m.DeletePostProjectsFunc(ctx, postID)
}

func (m *MockQuerier) InsertPostProject(ctx context.Context, arg database.InsertPostProjectParams) error {
	return m.InsertPostProjectFunc(ctx, arg)
}

func (m *MockQuerier) ListFeaturedPosts(ctx context.Context) ([]database.Post, error) {
	return m.ListFeaturedPostsFunc(ctx)
}

func (m *MockQuerier) ClearFeaturedPosts(ctx context.Context) error {
	return m.ClearFeaturedPostsFunc(ctx)
}

func (m *MockQuerier) SetFeaturedPost(ctx context.Context, arg database.SetFeaturedPostParams) error {
	return m.SetFeaturedPostFunc(ctx, arg)
}

func (m *MockQuerier) ListFeaturedProjects(ctx context.Context) ([]database.Project, error) {
	return m.ListFeaturedProjectsFunc(ctx)
}

func (m *MockQuerier) ClearFeaturedProjects(ctx context.Context) error {
	return m.ClearFeaturedProjectsFunc(ctx)
}

func (m *MockQuerier) SetFeaturedProject(ctx context.Context, arg database.SetFeaturedProjectParams) error {
	return m.SetFeaturedProjectFunc(ctx, arg)
}

var _ database.Querier = (*MockQuerier)(nil)
