package models

import (
	"database/sql"
	"fmt"
	"log"
	"sports-events-api/crypto"
	"sports-events-api/database"
	"strings"
	"time"
)

type Blogs struct {
	EncId       string    `json:"id"`
	Id          int64     `json:"-"`
	Title       string    `json:"title,omitempty" validate:"omitempty,min=2"`
	Image       string    `json:"image,omitempty"`
	Description string    `json:"description,omitempty" validate:"omitempty,min=2"`
	Content     string    `json:"content,omitempty"`
	Status      string    `json:"status,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

func InsertBlogs(BlogData *Blogs) (*Blogs, error) {
	// Set creation and update timestamps if they are not set
	if BlogData.CreatedAt.IsZero() {
		BlogData.CreatedAt = time.Now()
	}
	if BlogData.UpdatedAt.IsZero() {
		BlogData.UpdatedAt = time.Now()
	}

	BlogData.Status = "Active"

	// Insert the new blog into the database
	// var blogId int64
	query := `INSERT INTO blogs(title, image, description, content, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := database.DB.QueryRow(query, BlogData.Title, BlogData.Image, BlogData.Description, BlogData.Content, BlogData.Status, BlogData.CreatedAt, BlogData.UpdatedAt).Scan(&BlogData.Id)

	// Return an error if the insertion fails
	if err != nil {
		fmt.Printf("Error During Database Query : %v\n", err)
		return nil, fmt.Errorf("unable to create blogs : %v", err)
	}
	return BlogData, nil
}

func GetBlogs(search, sort, dir, status string, limit, offset int64) (int, []Blogs, error) {
	var blogs []Blogs
	args := []interface{}{limit, offset}
	query := `
        SELECT
            id, title, description, image, content, status, created_at, updated_at, COUNT(id) OVER() AS totalrecords
        FROM
            blogs
        WHERE status IN ('Active', 'Inactive')` // Only fetch Active and Inactive statuses

	// Add additional status filtering if provided
	if status != "" {
		statusValues := strings.Split(status, ",")
		statusPlaceholders := []string{}
		for _, s := range statusValues {
			statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(s))
		}
		query += fmt.Sprintf(" AND status IN (%s)", strings.Join(statusPlaceholders, ", "))
	}

	// Add search functionality
	if search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d)", len(args)+1)
		args = append(args, "%"+search+"%")
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $1 OFFSET $2", sort, dir)

	// Execute query
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying blogs: %v\n", err)
		return 0, nil, err
	}
	defer rows.Close()

	// Parse query results
	totalRecords := 0
	for rows.Next() {
		var blog Blogs
		if err := rows.Scan(
			&blog.Id,
			&blog.Title,
			&blog.Description,
			&blog.Image,
			&blog.Content,
			&blog.Status,
			&blog.CreatedAt,
			&blog.UpdatedAt,
			&totalRecords,
		); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			return 0, nil, err
		}
		blogImagePath := blog.Image
		defaultBlogLogoPath := "public/static/staticBlogImage.jpeg"

		if blog.Image == "" || !fileExists(blogImagePath) {
			blogImagePath = defaultBlogLogoPath
		}
		blog.Image = blogImagePath
		blog.EncId = crypto.NEncrypt(blog.Id)
		blogs = append(blogs, blog)
	}
	return totalRecords, blogs, nil
}

func DeleteBlogByID(id int64) (*Blogs, error) {
	// Check if the record exists in the database
	checkQuery := `SELECT id, status FROM blogs WHERE id = $1`
	var blog Blogs

	err := database.DB.QueryRow(checkQuery, id).Scan(&blog.Id, &blog.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("blog not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching blog: %v", err)
	}

	// If the blog is already marked as "Delete", return an error
	if blog.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Mark the blog as deleted by updating its status
	deleteQuery := `UPDATE blogs SET status = 'Delete', updated_at = $1 WHERE id = $2`
	_, err = database.DB.Exec(deleteQuery, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("error deleting blog: %v", err)
	}

	// Update the status to "Delete" and return the updated blog
	blog.Status = "Delete"
	blog.UpdatedAt = time.Now()

	return &blog, nil
}

func UpdateBlogStatusByID(blogId int64, status string) error {
	// Prepare the SQL query to update the status of the blog
	query := `UPDATE blogs SET status = $1 WHERE id = $2`

	// Execute the query
	_, err := database.DB.Exec(query, status, blogId)
	if err != nil {
		log.Printf("Error updating status. err :  %v\n", err)
		return fmt.Errorf("failed to update status")
	}

	return nil
}

func GetBlogStatusByID(blogId int64) (string, error) {
	var status string
	query := `SELECT status FROM blogs WHERE id = $1`
	err := database.DB.QueryRow(query, blogId).Scan(&status)
	if err != nil {
		log.Printf("Error fetching status for blog : %v\n", err)
		return "", fmt.Errorf("failed to fetch status")
	}
	return status, nil
}

func GetBlogById(id int64) (*Blogs, error) {
	// Query to fetch the blog details by ID
	query := `SELECT id, title, description, image, content, status, created_at, updated_at FROM blogs WHERE id=$1 AND status = 'Active'`

	var blog Blogs
	err := database.DB.QueryRow(query, id).Scan(
		&blog.Id,
		&blog.Title,
		&blog.Description,
		&blog.Image,
		&blog.Content,
		&blog.Status,
		&blog.CreatedAt,
		&blog.UpdatedAt,
	)

	// Check if the blog is marked as "Delete", return an error if so
	if blog.Status == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Handle cases where no record was found for the provided ID
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("blog is not found")
	} else if err != nil {
		return nil, fmt.Errorf("error fetching blog : %v", err)
	}

	blogImagePath := blog.Image
	defaultBlogLogoPath := "public/static/staticBlogImage.jpeg"

	if blog.Image == "" || !fileExists(blogImagePath) {
		blogImagePath = defaultBlogLogoPath
	}
	blog.Image = blogImagePath

	// Return the fetched game type
	return &blog, nil
}

func UpdateBlog(blogData *Blogs) (*Blogs, error) {
	// Check if the record exists
	checkQuery := `SELECT id, status FROM blogs WHERE id = $1`
	var existingID int64
	var existingStatus string

	err := database.DB.QueryRow(checkQuery, blogData.Id).Scan(&existingID, &existingStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("blog does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("database error while checking blog: %v", err)
	}

	if existingStatus == "Delete" {
		return nil, fmt.Errorf("no data found")
	}

	// Proceed with the update
	blogData.UpdatedAt = time.Now()

	updateQuery := `
        UPDATE blogs 
        SET title = $1, description = $2, image = $3, content = $4, updated_at = $5 
        WHERE id = $6
        RETURNING id, title, description, image, content, status, created_at, updated_at`

	var updatedBlog Blogs
	err = database.DB.QueryRow(updateQuery, blogData.Title, blogData.Description, blogData.Image, blogData.Content, blogData.UpdatedAt, blogData.Id).Scan(
		&updatedBlog.Id,
		&updatedBlog.Title,
		&updatedBlog.Description,
		&updatedBlog.Image,
		&updatedBlog.Content,
		&updatedBlog.Status,
		&updatedBlog.CreatedAt,
		&updatedBlog.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("database error during update: %v", err)
	}

	return &updatedBlog, nil
}
