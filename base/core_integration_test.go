package base

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test models
type User struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Email     string
	Profile   Profile   // Has One
	Posts     []Post    // Has Many
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (User) TableName() string {
	return "dummy_users"
}

func (User) PrimaryKey() string {
	return "id"
}

type Profile struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Bio       string
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Profile) TableName() string {
	return "dummy_profiles"
}

func (Profile) PrimaryKey() string {
	return "id"
}

type Post struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Title     string
	Content   string
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Post) TableName() string {
	return "dummy_posts"
}

func (Post) PrimaryKey() string {
	return "id"
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3306"
	}
	dbName := os.Getenv("MYSQL_DATABASE")
	if dbName == "" {
		dbName = "demo"
	}
	username := os.Getenv("MYSQL_USERNAME")
	if username == "" {
		username = "root"
	}
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		t.Skip("MYSQL_PASSWORD environment variable not set")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username,
		password,
		host,
		port,
		dbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate the test models
	err = db.AutoMigrate(&User{}, &Profile{}, &Post{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func cleanupDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Clean up existing data
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	db.Exec("TRUNCATE TABLE dummy_users")
	db.Exec("TRUNCATE TABLE dummy_profiles")
	db.Exec("TRUNCATE TABLE dummy_posts")
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

func TestCRUDOperations(t *testing.T) {
	db := setupTestDB(t)

	// Ensure cleanup after test
	t.Cleanup(func() {
		cleanupDB(t, db)
	})

	// Clean before test
	cleanupDB(t, db)

	baseRepo := NewBaseGorm[User, uint](db)
	ctx := context.Background()

	t.Run("Create Operations", func(t *testing.T) {
		// Test single create
		user := &User{
			Name:  "Test User",
			Email: "test@example.com",
		}
		createdUser, err := baseRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if createdUser.ID == 0 {
			t.Error("Expected user ID to be set after creation")
		}

		// Test multiple create
		users := []*User{
			{Name: "User 1", Email: "user1@example.com"},
			{Name: "User 2", Email: "user2@example.com"},
		}
		createdUsers, count, err := baseRepo.CreateMultiple(ctx, users)
		if err != nil {
			t.Fatalf("Failed to create multiple users: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 users to be created, got %d", count)
		}
		if len(createdUsers) != 2 {
			t.Errorf("Expected 2 users in result, got %d", len(createdUsers))
		}
	})

	t.Run("Read Operations", func(t *testing.T) {
		// Create test data
		user := &User{
			Name:  "Read Test User",
			Email: "read@example.com",
		}
		user, err := baseRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Test Detail
		foundUser, err := baseRepo.Detail(ctx, user.ID)
		if err != nil {
			t.Errorf("Failed to get user detail: %v", err)
		}
		if foundUser.ID != user.ID {
			t.Errorf("Expected user ID %d, got %d", user.ID, foundUser.ID)
		}

		// Test Wheres
		where := Where{
			Name:  "email",
			Value: "read@example.com",
		}
		foundUser, err = baseRepo.Wheres(ctx, []Where{where})
		if err != nil {
			t.Errorf("Failed to find user with where clause: %v", err)
		}
		if foundUser.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, foundUser.Email)
		}

		// Test WheresList
		users, err := baseRepo.WheresList(ctx, nil, []Where{where})
		if err != nil {
			t.Errorf("Failed to get users list with where clause: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}

		// Test List with pagination
		allUsers, paginator, err := baseRepo.List(ctx, 1, 10, nil, nil)
		if err != nil {
			t.Errorf("Failed to get users list: %v", err)
		}
		log.Printf("allUsers: %+v, paginator: %+v\n", allUsers, paginator)
		if paginator.Total < 1 {
			t.Error("Expected at least 1 user in total count")
		}

		// Test ListCustom
		customUsers, customPaginator, err := baseRepo.ListCustom(ctx, 1, 10, nil, nil, func(db *gorm.DB) *gorm.DB {
			return db.Model(&User{}).Where("email LIKE ?", "%@example.com")
		})
		if err != nil {
			t.Errorf("Failed to get custom users list: %v", err)
		}
		if len(customUsers) < 1 {
			t.Error("Expected at least 1 user in custom list")
		}
		if customPaginator.Total < 1 {
			t.Error("Expected at least 1 user in custom total count")
		}
	})

	t.Run("Update Operations", func(t *testing.T) {
		// Create test data
		user := &User{
			Name:  "Update Test User",
			Email: "update@example.com",
		}
		user, err := baseRepo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		// Test Update
		user.Name = "Updated Name"
		rowsAffected, err := baseRepo.Update(ctx, user, []string{"name"})
		if err != nil {
			t.Errorf("Failed to update user: %v", err)
		}
		if rowsAffected != 1 {
			t.Errorf("Expected 1 row affected, got %d", rowsAffected)
		}

		// Verify update
		updatedUser, err := baseRepo.Detail(ctx, user.ID)
		if err != nil {
			t.Errorf("Failed to get updated user: %v", err)
		}
		if updatedUser.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", updatedUser.Name)
		}

		// Test UpdateWhere
		where := Where{
			Name:  "email",
			Value: "update@example.com",
		}
		values := map[string]interface{}{
			"name": "Updated Via Where",
		}
		rowsAffected, err = baseRepo.UpdateWhere(ctx, []Where{where}, values)
		if err != nil {
			t.Errorf("Failed to update user with where clause: %v", err)
		}
		if rowsAffected != 1 {
			t.Errorf("Expected 1 row affected in UpdateWhere, got %d", rowsAffected)
		}

		// Test Upsert
		upsertUser := &User{
			ID:    user.ID,
			Name:  "Upserted Name",
			Email: "upsert@example.com",
		}
		rowsAffected, err = baseRepo.Upsert(ctx, upsertUser, []string{"name", "email"})
		if err != nil {
			t.Errorf("Failed to upsert user: %v", err)
		}
		if rowsAffected != 2 { // MySQL returns 2 for update with ON DUPLICATE KEY UPDATE
			t.Errorf("Expected 2 rows affected in Upsert (MySQL behavior), got %d", rowsAffected)
		}

		// Verify the upsert
		updatedUser, err = baseRepo.Detail(ctx, user.ID)
		if err != nil {
			t.Errorf("Failed to get upserted user: %v", err)
		}
		if updatedUser.Name != "Upserted Name" {
			t.Errorf("Expected name 'Upserted Name', got '%s'", updatedUser.Name)
		}
		if updatedUser.Email != "upsert@example.com" {
			t.Errorf("Expected email 'upsert@example.com', got '%s'", updatedUser.Email)
		}
	})
}

func TestAssociations(t *testing.T) {
	// Prevent parallel test execution since we're dealing with a shared database
	db := setupTestDB(t)

	// Ensure cleanup after test
	t.Cleanup(func() {
		cleanupDB(t, db)
	})

	// Clean before test
	cleanupDB(t, db)

	baseRepo := NewBaseGorm[User, uint](db)
	ctx := context.Background()

	// Create a test user
	user := &User{
		Name:  "Test User",
		Email: "test@example.com",
	}
	user, err := baseRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("Profile Association Tests", func(t *testing.T) {
		// Test AppendAssociation - Profile (Has One)
		profile := &Profile{
			Bio: "Test Bio",
		}
		if err := baseRepo.AppendAssociation(ctx, user, "Profile", profile); err != nil {
			t.Fatalf("Failed to append profile: %v", err)
		}

		// Test CountAssociation - Profile
		profileCount := baseRepo.CountAssociation(ctx, user, "Profile")
		if profileCount != 1 {
			t.Errorf("Profile association count mismatch: expected 1, got %d", profileCount)
		}
	})

	t.Run("Posts Association Tests", func(t *testing.T) {
		// Test AppendAssociation - Posts (Has Many)
		posts := []*Post{
			{Title: "Post 1", Content: "Content 1"},
			{Title: "Post 2", Content: "Content 2"},
		}
		if err := baseRepo.AppendAssociation(ctx, user, "Posts", posts); err != nil {
			t.Fatalf("Failed to append posts: %v", err)
		}

		// Test CountAssociation - Posts
		postsCount := baseRepo.CountAssociation(ctx, user, "Posts")
		if postsCount != 2 {
			t.Errorf("Posts association count mismatch: expected 2, got %d", postsCount)
		}

		// Test FindAssociation - Posts
		var foundPosts []*Post
		if err := baseRepo.FindAssociation(ctx, user, "Posts", &foundPosts); err != nil {
			t.Fatalf("Failed to find posts: %v", err)
		}
		if len(foundPosts) != 2 {
			t.Errorf("Found posts count mismatch: expected 2, got %d", len(foundPosts))
		}

		// Test ReplaceAssociation - Posts
		newPosts := []*Post{
			{Title: "New Post", Content: "New Content"},
		}
		if err := baseRepo.ReplaceAssociation(ctx, user, "Posts", newPosts); err != nil {
			t.Fatalf("Failed to replace posts: %v", err)
		}

		// Verify replacement
		postsCount = baseRepo.CountAssociation(ctx, user, "Posts")
		if postsCount != 1 {
			t.Errorf("Posts count after replacement mismatch: expected 1, got %d", postsCount)
		}

		// Test ClearAssociation - Posts
		if err := baseRepo.ClearAssociation(ctx, user, "Posts"); err != nil {
			t.Fatalf("Failed to clear posts: %v", err)
		}

		// Verify clearing
		postsCount = baseRepo.CountAssociation(ctx, user, "Posts")
		if postsCount != 0 {
			t.Errorf("Posts count after clearing mismatch: expected 0, got %d", postsCount)
		}
	})
}

func TestTransactionWithAssociations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupDB(t, db)

	ctx := context.Background()
	baseRepo := NewBaseGorm[User, uint](db)

	t.Run("Create_With_Associations", func(t *testing.T) {
		// Create user with profile and posts in a transaction
		user := &User{
			Name:  "Transaction Test User",
			Email: "transaction@example.com",
			Profile: Profile{
				Bio: "Test user bio",
			},
			Posts: []Post{
				{
					Title:   "First Post",
					Content: "First post content",
				},
				{
					Title:   "Second Post",
					Content: "Second post content",
				},
			},
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			txRepo := NewBaseGorm[User, uint](tx)
			user, err := txRepo.Create(ctx, user)
			log.Printf("user: %+v\n", user)
			if err != nil {
				return fmt.Errorf("failed to create user: %v", err)
			}

			// Verify user was created
			if user.ID == 0 {
				return fmt.Errorf("user ID should not be 0 after creation")
			}

			// Verify profile was created
			if user.Profile.ID == 0 || user.Profile.UserID != user.ID {
				return fmt.Errorf("profile was not created properly")
			}

			// Verify posts were created
			for _, post := range user.Posts {
				if post.ID == 0 || post.UserID != user.ID {
					return fmt.Errorf("post was not created properly")
				}
			}

			return nil
		})

		if err != nil {
			t.Errorf("Transaction failed: %v", err)
		}

		// Verify data persisted after transaction
		savedUser, err := baseRepo.Detail(ctx, user.ID)
		if err != nil {
			t.Errorf("Failed to get user after transaction: %v", err)
		}

		// Load associations
		err = db.Model(&savedUser).Association("Profile").Find(&savedUser.Profile)
		if err != nil {
			t.Errorf("Failed to load profile: %v", err)
		}
		err = db.Model(&savedUser).Association("Posts").Find(&savedUser.Posts)
		if err != nil {
			t.Errorf("Failed to load posts: %v", err)
		}

		// Verify all data
		if savedUser.Name != user.Name {
			t.Errorf("Expected user name %s, got %s", user.Name, savedUser.Name)
		}
		if savedUser.Profile.Bio != user.Profile.Bio {
			t.Errorf("Expected profile bio %s, got %s", user.Profile.Bio, savedUser.Profile.Bio)
		}
		if len(savedUser.Posts) != len(user.Posts) {
			t.Errorf("Expected %d posts, got %d", len(user.Posts), len(savedUser.Posts))
		}
	})

	t.Run("Transaction_Rollback", func(t *testing.T) {
		var initialCount int64
		db.Model(&User{}).Count(&initialCount)

		user := &User{
			Name:  "Rollback Test User",
			Email: "rollback@example.com",
			Profile: Profile{
				Bio: "This should be rolled back",
			},
			Posts: []Post{
				{
					Title:   "Rollback Post",
					Content: "This should be rolled back",
				},
			},
		}

		// Execute transaction that will be rolled back
		err := db.Transaction(func(tx *gorm.DB) error {
			txRepo := NewBaseGorm[User, uint](tx)
			if _, err := txRepo.Create(ctx, user); err != nil {
				return err
			}
			// Force a rollback
			return fmt.Errorf("forcing rollback")
		})

		if err == nil {
			t.Error("Expected transaction to fail")
		}

		// Verify nothing was persisted
		var finalCount int64
		db.Model(&User{}).Count(&finalCount)
		if finalCount != initialCount {
			t.Errorf("Expected user count to remain %d, got %d", initialCount, finalCount)
		}
	})
}
