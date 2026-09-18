package middleware

// func GatewayAuthMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		expected := os.Getenv("GATEWAY_SECRET")
// 		if expected == "" {
// 			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
// 				"error": "GATEWAY_SECRET not configured",
// 			})
// 			return
// 		}

// 		if c.GetHeader("X-Gateway-Auth") != expected {
// 			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
// 				"error": "unauthorized gateway",
// 			})
// 			return
// 		}

// 		c.Next()
// 	}
// }
