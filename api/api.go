package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetIdeas(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "getIdeas Called"})
}

func GetIdeaById(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "getIdeaById " + id + " Called"})
}

func AddIdea(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "addIdea Called"})
}

func UpdateIdea(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "updateIdea Called"})
}

func DeleteIdea(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "deleteIdea " + id + " Called"})
}

func Options(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "options Called"})
}
