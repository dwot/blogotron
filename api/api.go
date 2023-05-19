package api

import (
	"github.com/gin-gonic/gin"
	"golang/models"
	"golang/util"
	"net/http"
	"strconv"
)

func GetIdeas(c *gin.Context) {
	ideas, err := models.GetIdeas()
	util.Logger.Error().Err(err).Msg("GetIdeas")

	if ideas == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Records Found"})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"data": ideas})
	}
}

func GetIdeaById(c *gin.Context) {
	id := c.Param("id")
	idea, err := models.GetIdeaById(id)
	util.Logger.Error().Err(err).Msg("GetIdeaById")
	// if the name is blank we can assume nothing is found
	if idea.IdeaText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Records Found"})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"data": idea})
	}
}

func AddIdea(c *gin.Context) {
	var json models.Idea

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	success, err := models.AddIdea(json)

	if success {
		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}
}

func UpdateIdea(c *gin.Context) {
	var json models.Idea

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ideaId, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
	}

	success, err := models.UpdateIdea(json, ideaId)

	if success {
		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}
}

func DeleteIdea(c *gin.Context) {
	ideaId, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
	}

	success, err := models.DeleteIdea(ideaId)

	if success {
		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}
}

func Options(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "options Called"})
}
