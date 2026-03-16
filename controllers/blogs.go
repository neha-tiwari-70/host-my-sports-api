package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sports-events-api/crypto"
	"sports-events-api/models"
	"sports-events-api/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateBlogs(c *gin.Context) {
	// Parse multipart form
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		utils.HandleError(c, "Invalid form data", err)
		return
	}

	var BlogData models.Blogs
	BlogData.Title = c.PostForm("title")
	BlogData.Description = c.PostForm("description")
	BlogData.Content = c.PostForm("content")

	file, header, err := c.Request.FormFile("image")
	if err == nil && file != nil {
		defer file.Close()
		fileExt := filepath.Ext(header.Filename)
		safeName := strings.ReplaceAll("blog", " ", "_")
		fileName := fmt.Sprintf("qr_%s_%d%s", safeName, time.Now().Unix(), fileExt)
		uploadPath := filepath.Join("public/uploads", fileName)

		out, err := os.Create(uploadPath)
		if err != nil {
			utils.HandleError(c, "Failed to save image", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			utils.HandleError(c, "Failed to save image", err)
			return
		}

		BlogData.Image = uploadPath
	}

	createdBlog, err := models.InsertBlogs(&BlogData)
	if err != nil {
		utils.HandleError(c, "Unable to create blog.", err)
		return
	}

	createdBlog.EncId = crypto.NEncrypt(createdBlog.Id)

	utils.HandleSuccess(c, "Blog created successfully.", createdBlog)
}

func GetAllBlogs(c *gin.Context) {
	// Extract query parameters for pagination, search, sorting, and status
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	sort := c.DefaultQuery("sort", "created_at")
	dir := c.DefaultQuery("dir", "DESC")
	status := c.Query("status") // Fetch status from query params
	offset := (page - 1) * limit

	// Fetch data from the model with status filtering
	totalRecords, blogs, err := models.GetBlogs(search, sort, dir, status, int64(limit), int64(offset))
	if err != nil {
		utils.HandleError(c, "Failed to fetch blogs.", err)
		return
	}

	// Respond with paginated and filtered data
	utils.HandleSuccess(c, "Fetched all blogs successfully.", gin.H{
		"totalRecords": totalRecords,
		"blogs":        blogs,
	})
}

func DeleteBlog(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	_, err := models.DeleteBlogByID(decryptedId)
	if err != nil {
		utils.HandleError(c, err.Error())
		return
	}

	utils.HandleSuccess(c, "Blog deleted successfully.")
}

func UpdateBlogStatus(c *gin.Context) {
	// Extract the blog ID from the URL parameter
	decBlogId := DecryptParamId(c, "id", true)
	if decBlogId == 0 {
		return
	}

	// Fetch the current status from the database
	currentStatus, err := models.GetBlogStatusByID(decBlogId)
	if err != nil {
		utils.HandleError(c, "Failed to fetch current status.", err)
		return
	}

	// Determine the new status based on the current status
	newStatus := "Inactive"
	if currentStatus == "Inactive" {
		newStatus = "Active"
	}

	// Update the status in the database
	err = models.UpdateBlogStatusByID(decBlogId, newStatus)
	if err != nil {
		utils.HandleError(c, "Failed to update status.", err)
		return
	}

	// Respond with success
	utils.HandleSuccess(c, "Blog status updated successfully.")
}

func GetBlogById(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	blog, err := models.GetBlogById(decryptedId)
	if err != nil {
		utils.HandleError(c, err.Error())
		return
	}
	blog.EncId = crypto.NEncrypt(blog.Id)
	utils.HandleSuccess(c, "Blog data fetched successfully", blog)
}

func UpdateBlogById(c *gin.Context) {
	decryptedId := DecryptParamId(c, "id", true)
	if decryptedId == 0 {
		return
	}

	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		utils.HandleError(c, "Invalid form data", err)
		return
	}

	var BlogData models.Blogs
	BlogData.Title = c.PostForm("title")
	BlogData.Description = c.PostForm("description")
	BlogData.Content = c.PostForm("content")

	existingBlogInfo, _ := models.GetBlogById(decryptedId)

	file, header, err := c.Request.FormFile("image")
	if err == nil && file != nil {
		defer file.Close()
		fileExt := filepath.Ext(header.Filename)
		safeName := strings.ReplaceAll("blog", " ", "_")
		fileName := fmt.Sprintf("qr_%s_%d%s", safeName, time.Now().Unix(), fileExt)
		uploadPath := filepath.Join("public/uploads", fileName)

		out, err := os.Create(uploadPath)
		if err != nil {
			utils.HandleError(c, "Failed to save image", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			utils.HandleError(c, "Failed to save image", err)
			return
		}

		BlogData.Image = uploadPath
	} else {
		// No new file, keep the existing one
		if existingBlogInfo != nil {
			BlogData.Image = existingBlogInfo.Image
		}
	}

	// Populate the ID in the games_types struct
	BlogData.Id = decryptedId

	// Call the model function to update the games type
	updatedBlog, err := models.UpdateBlog(&BlogData)
	if err != nil {
		utils.HandleError(c, "Unable to update blog.", err)
		return
	}

	// Encrypt the updated ID for the response
	updatedBlog.EncId = crypto.NEncrypt(updatedBlog.Id)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Blog updated successfully.",
		"data":    updatedBlog,
	})
}
