# Class Diagram: Gateway Service (CR-011)

Diagram kelas untuk layanan terpadu `gateway` yang mengintegrasikan API Gateway Reverse Proxy dan In-Process Auth Service.

```mermaid
classDiagram
    class GatewayServer {
        -Router *gin.Engine
        -ProxyHandler *ProxyHandler
        -AuthModule *AuthModule
        +Start(port string) error
        +Shutdown(ctx context.Context) error
    }

    class ProxyHandler {
        -UpstreamURL string
        -HMACSigner *HMACSigner
        -JWKSValidator *JWKSValidator
        +ServeHTTP(c *gin.Context)
        +ForwardToPMB(c *gin.Context)
        +ForwardToOCR(c *gin.Context)
    }

    class HMACSigner {
        -SecretKey []byte
        +Sign(req *http.Request, userID string, roles []string) error
        +Verify(signature string, body []byte) bool
    }

    class JWKSValidator {
        -PublicKey *rsa.PublicKey
        -KeyID string
        +ValidateToken(tokenStr string) (*Claims, error)
        +GetPublicKeyJWKS() map[string]interface{}
    }

    class AuthController {
        -authSvc AuthService
        -tokenSvc TokenService
        +Login(c *gin.Context)
        +Register(c *gin.Context)
        +RefreshToken(c *gin.Context)
        +ForgotPassword(c *gin.Context)
        +ResetPassword(c *gin.Context)
        +VerifyOTP(c *gin.Context)
    }

    class AuthService {
        -userRepo UserRepository
        -roleRepo RoleRepository
        -auditRepo AuditLogRepository
        +Authenticate(email, password string) (*User, error)
        +CreateUser(req RegisterRequest) (*User, error)
        +AssignRole(userID, roleID uuid.UUID) error
    }

    class TokenService {
        -privateKey *rsa.PrivateKey
        -issuer string
        +GeneratePair(user *User) (*TokenPair, error)
        +ValidateRefreshToken(token string) (*User, error)
        +RevokeToken(tokenID uuid.UUID) error
    }

    class UserRepository {
        <<interface>>
        +FindByID(id uuid.UUID) (*User, error)
        +FindByEmail(email string) (*User, error)
        +Create(user *User) error
        +Update(user *User) error
    }

    class User {
        +UUID ID
        +string Name
        +string Email
        +string PasswordHash
        +string Status
        +time.Time CreatedAt
    }

    class Role {
        +UUID ID
        +string Name
        +string Description
    }

    class Permission {
        +UUID ID
        +string Code
        +string Module
    }

    GatewayServer --> ProxyHandler
    GatewayServer --> AuthController
    ProxyHandler --> HMACSigner
    ProxyHandler --> JWKSValidator
    AuthController --> AuthService
    AuthController --> TokenService
    AuthService --> UserRepository
    User "1" -- "*" Role : has
    Role "1" -- "*" Permission : contains
```
