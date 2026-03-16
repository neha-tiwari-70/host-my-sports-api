package controllers

import (
	"fmt"
	"sports-events-api/crypto"
	"sports-events-api/utils"

	"github.com/gin-gonic/gin"
)

func DecryptParamId(c *gin.Context, ParamName string, required bool) int64 {
	paramId := int64(0)
	var err error
	// Step 1: Extract encrypted paramId
	EncId, exists := c.Params.Get(ParamName)
	if !exists && required {
		utils.HandleError(c, fmt.Sprintf("Invalid request-->missing %v param", ParamName))
		return paramId
	} else if !exists {
		return paramId
	}

	// Step 3: Decrypt paramId
	paramId, err = crypto.NDecrypt(EncId)
	if err != nil {
		utils.HandleError(c, "Decryption Error", fmt.Errorf("error decrypting EncId(value:'%v')->%v", EncId, err))
		return paramId
	}
	return paramId
}
