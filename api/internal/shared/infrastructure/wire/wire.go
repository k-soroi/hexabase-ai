//go:build wireinject
// +build wireinject

package wire

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/wire"
	"github.com/hexabase/hexabase-ai/api/internal/shared/api/handlers"

	// Application domain - using new package-by-feature structure
	"github.com/hexabase/hexabase-ai/api/internal/application/domain"
	applicationHandler "github.com/hexabase/hexabase-ai/api/internal/application/handler"
	applicationRepo "github.com/hexabase/hexabase-ai/api/internal/application/repository"
	applicationSvc "github.com/hexabase/hexabase-ai/api/internal/application/service"

	// Auth domain
	internalAuth "github.com/hexabase/hexabase-ai/api/internal/auth"
	authDomain "github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	authHandler "github.com/hexabase/hexabase-ai/api/internal/auth/handler"
	authRepo "github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	authSvc "github.com/hexabase/hexabase-ai/api/internal/auth/service"

	// Organization domain
	orgHandler "github.com/hexabase/hexabase-ai/api/internal/organization/handler"
	orgRepo "github.com/hexabase/hexabase-ai/api/internal/organization/repository"
	orgSvc "github.com/hexabase/hexabase-ai/api/internal/organization/service"

	// Project domain
	projectDomain "github.com/hexabase/hexabase-ai/api/internal/project/domain"
	projectHandler "github.com/hexabase/hexabase-ai/api/internal/project/handler"
	projectRepo "github.com/hexabase/hexabase-ai/api/internal/project/repository"
	projectSvc "github.com/hexabase/hexabase-ai/api/internal/project/service"

	// Workspace domain
	workspaceDomain "github.com/hexabase/hexabase-ai/api/internal/workspace/domain"
	workspaceHandler "github.com/hexabase/hexabase-ai/api/internal/workspace/handler"
	workspaceRepo "github.com/hexabase/hexabase-ai/api/internal/workspace/repository"
	workspaceSvc "github.com/hexabase/hexabase-ai/api/internal/workspace/service"

	// Node domain (migrated)
	nodeDomain "github.com/hexabase/hexabase-ai/api/internal/node/domain"
	nodeHandler "github.com/hexabase/hexabase-ai/api/internal/node/handler"
	nodeRepo "github.com/hexabase/hexabase-ai/api/internal/node/repository"
	nodeService "github.com/hexabase/hexabase-ai/api/internal/node/service"

	// Monitoring domain (migrated)
	monitoringDomain "github.com/hexabase/hexabase-ai/api/internal/monitoring/domain"
	monitoringHandler "github.com/hexabase/hexabase-ai/api/internal/monitoring/handler"
	monitoringRepo "github.com/hexabase/hexabase-ai/api/internal/monitoring/repository"
	monitoringSvc "github.com/hexabase/hexabase-ai/api/internal/monitoring/service"

	// AIOps (migrated)
	aiopsDomain "github.com/hexabase/hexabase-ai/api/internal/aiops/domain"
	aiopsHandler "github.com/hexabase/hexabase-ai/api/internal/aiops/handler"
	aiopsRepo "github.com/hexabase/hexabase-ai/api/internal/aiops/repository"
	aiopsSvc "github.com/hexabase/hexabase-ai/api/internal/aiops/service"

	// Backup (migrated)
	backupDomain "github.com/hexabase/hexabase-ai/api/internal/backup/domain"
	backupHandler "github.com/hexabase/hexabase-ai/api/internal/backup/handler"
	backupRepo "github.com/hexabase/hexabase-ai/api/internal/backup/repository"
	backupSvc "github.com/hexabase/hexabase-ai/api/internal/backup/service"

	// Billing (migrated)
	billingDomain "github.com/hexabase/hexabase-ai/api/internal/billing/domain"
	billingHandler "github.com/hexabase/hexabase-ai/api/internal/billing/handler"
	billingRepo "github.com/hexabase/hexabase-ai/api/internal/billing/repository"
	billingSvc "github.com/hexabase/hexabase-ai/api/internal/billing/service"

	// Logs (migrated)
	logsDomain "github.com/hexabase/hexabase-ai/api/internal/logs/domain"
	logsRepo "github.com/hexabase/hexabase-ai/api/internal/logs/repository"
	logsSvc "github.com/hexabase/hexabase-ai/api/internal/logs/service"

	// Function (migrated)
	functionDomain "github.com/hexabase/hexabase-ai/api/internal/function/domain"
	functionHandler "github.com/hexabase/hexabase-ai/api/internal/function/handler"
	functionRepo "github.com/hexabase/hexabase-ai/api/internal/function/repository"
	functionSvc "github.com/hexabase/hexabase-ai/api/internal/function/service"

	// CICD (migrated)
	cicdDomain "github.com/hexabase/hexabase-ai/api/internal/cicd/domain"
	cicdHandler "github.com/hexabase/hexabase-ai/api/internal/cicd/handler"
	cicdRepo "github.com/hexabase/hexabase-ai/api/internal/cicd/repository"
	cicdSvc "github.com/hexabase/hexabase-ai/api/internal/cicd/service"

	// Legacy repositories that haven't been migrated yet
	proxmoxRepo "github.com/hexabase/hexabase-ai/api/internal/node/repository/proxmox"
	k8sRepo "github.com/hexabase/hexabase-ai/api/internal/shared/kubernetes/repository"

	"github.com/hexabase/hexabase-ai/api/internal/helm"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	"gorm.io/gorm"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// Updated wire sets for migrated packages
var ApplicationSet = wire.NewSet(
	applicationRepo.NewPostgresRepository,
	applicationRepo.NewKubernetesRepository,
	applicationSvc.NewService,
	applicationHandler.NewApplicationHandler,
)

var AuthSet = wire.NewSet(
	ProvideRedisClient,
	authRepo.NewPostgresRepository,
	authRepo.NewRedisAuthRepository,
	authRepo.NewTokenHashRepository, // 追加
	authRepo.NewCompositeRepository,
	authRepo.NewOAuthRepository,
	authRepo.NewKeyRepository,
	authRepo.NewSessionLimiterRepository,
	ProvideTokenManager,
	ProvideTokenDomainService,
	ProvideDefaultTokenExpiry,
	authSvc.NewSessionManager,
	authSvc.NewService,
	authHandler.NewHandler,
)

var OrganizationSet = wire.NewSet(
	orgRepo.NewPostgresRepository,
	orgRepo.NewAuthRepositoryAdapter,
	orgRepo.NewBillingRepositoryAdapter,
	orgSvc.NewService,
	orgHandler.NewHandler,
)

var ProjectSet = wire.NewSet(
	projectRepo.NewPostgresRepository,
	projectRepo.NewKubernetesRepository,
	projectSvc.NewService,
	projectHandler.NewHandler,
)

var WorkspaceSet = wire.NewSet(
	workspaceRepo.NewPostgresRepository,
	workspaceRepo.NewKubernetesRepository,
	workspaceRepo.NewAuthRepositoryAdapter,
	workspaceSvc.NewService,
	workspaceHandler.NewHandler,
)

// Legacy wire sets for packages that haven't been migrated yet
var BackupSet = wire.NewSet(
	backupRepo.NewPostgresRepository, 
	ProvideBackupProxmoxRepository, 
	ProvideBackupEncryptionKey,
	backupSvc.NewService,
	backupHandler.NewHandler,
)

var BillingSet = wire.NewSet(
	billingRepo.NewPostgresRepository,
	ProvideStripeRepository,
	billingSvc.NewService,
	billingHandler.NewHandler,
)

var MonitoringSet = wire.NewSet(
	monitoringRepo.NewPostgresRepository,
	k8sRepo.NewKubernetesRepository,
	monitoringSvc.NewService,
	monitoringHandler.NewHandler,
)

var NodeSet = wire.NewSet(
	nodeRepo.NewPostgresRepository,
	ProvideNodeRepository,
	ProvideProxmoxRepository,
	ProvideProxmoxRepositoryInterface,
	nodeService.NewService,
	ProvideNodeService,
	nodeHandler.NewHandler,
)

var CICDSet = wire.NewSet(
	cicdRepo.NewPostgresRepository,
	ProvideCICDProviderFactory,
	ProvideCICDCredentialManager,
	cicdSvc.NewService,
	cicdHandler.NewHandler,
)

var FunctionSet = wire.NewSet(
	ProvideSQLDB,
	functionRepo.NewPostgresRepository,
	ProvideFunctionRepository,
	functionRepo.NewProviderFactory,
	ProvideFunctionProviderFactory,
	functionSvc.NewService,
	ProvideFunctionService,
	functionHandler.NewHandler,
)

var HelmSet = wire.NewSet(helm.NewService)

var AIOpsSet = wire.NewSet(
	aiopsRepo.NewPostgresRepository,
	ProvideOllamaService,
	aiopsSvc.NewService,
	ProvideAIOpsAdapter,
	aiopsHandler.NewHandler,
	aiopsHandler.NewGinHandler,
	ProvideAIOpsServiceURL,
	ProvideAIOpsProxyHandler,
)

var LogSet = wire.NewSet(
	ProvideClickHouseConn,
	logsRepo.NewClickHouseRepository,
	logsSvc.NewLogService,
)

var InternalSet = wire.NewSet(ProvideInternalHandler)

type App struct {
	ApplicationHandler  *applicationHandler.ApplicationHandler
	AuthHandler        *authHandler.Handler
	BackupHandler      *backupHandler.Handler
	BillingHandler     *billingHandler.Handler
	CICDHandler        *cicdHandler.Handler
	MonitoringHandler  *monitoringHandler.Handler
	NodeHandler        *nodeHandler.Handler
	OrganizationHandler *orgHandler.Handler
	ProjectHandler     *projectHandler.Handler
	WorkspaceHandler   *workspaceHandler.Handler
	FunctionHandler    *functionHandler.Handler
	AIOpsHandler       *aiopsHandler.Handler
	AIOpsGinHandler    *aiopsHandler.GinHandler
	AIOpsProxyHandler  *aiopsHandler.AIOpsProxyHandler
	InternalHandler    *handlers.InternalHandler
	LogSvc             logsDomain.Service
}

func NewApp(
	appH *applicationHandler.ApplicationHandler,
	authH *authHandler.Handler,
	backupH *backupHandler.Handler,
	billH *billingHandler.Handler,
	cicdH *cicdHandler.Handler,
	monH *monitoringHandler.Handler,
	nodeH *nodeHandler.Handler,
	orgH *orgHandler.Handler,
	projH *projectHandler.Handler,
	workH *workspaceHandler.Handler,
	funcH *functionHandler.Handler,
	aiopsH *aiopsHandler.Handler,
	aiopsGinH *aiopsHandler.GinHandler,
	aiopsProxyH *aiopsHandler.AIOpsProxyHandler,
	internalHandler *handlers.InternalHandler,
	logSvc logsDomain.Service,
) *App {
	return &App{
		ApplicationHandler:  appH,
		AuthHandler:        authH,
		BackupHandler:      backupH,
		BillingHandler:     billH,
		CICDHandler:        cicdH,
		MonitoringHandler:  monH,
		NodeHandler:        nodeH,
		OrganizationHandler: orgH,
		ProjectHandler:     projH,
		WorkspaceHandler:   workH,
		FunctionHandler:    funcH,
		AIOpsHandler:       aiopsH,
		AIOpsGinHandler:    aiopsGinH,
		AIOpsProxyHandler:  aiopsProxyH,
		InternalHandler:    internalHandler,
		LogSvc:             logSvc,
	}
}

type StripeAPIKey string
type StripeWebhookSecret string
type AIOpsServiceURL string
type CICDNamespace string
type BackupEncryptionKey string

func ProvideOAuthProviderConfigs(cfg *config.Config) map[string]*authRepo.ProviderConfig {
	providers := make(map[string]*authRepo.ProviderConfig)
	if cfg.Auth.ExternalProviders == nil {
		return providers
	}
	for name, p := range cfg.Auth.ExternalProviders {
		providers[name] = &authRepo.ProviderConfig{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  p.RedirectURL,
			Scopes:       p.Scopes,
			AuthURL:      p.AuthURL,
			TokenURL:     p.TokenURL,
		}
	}
	return providers
}

func ProvideStripeAPIKey(cfg *config.Config) StripeAPIKey { return StripeAPIKey(cfg.Stripe.APIKey) }
func ProvideStripeWebhookSecret(cfg *config.Config) StripeWebhookSecret { return StripeWebhookSecret(cfg.Stripe.WebhookSecret) }
func ProvideAIOpsServiceURL(cfg *config.Config) (AIOpsServiceURL, error) { 
	if cfg.AIOps.URL != "" { 
		return AIOpsServiceURL(cfg.AIOps.URL), nil 
	}
	return AIOpsServiceURL("http://ai-ops-service.ai-ops.svc.cluster.local:8000"), nil 
}
func ProvideStripeRepository(apiKey StripeAPIKey, webhookSecret StripeWebhookSecret) billingDomain.StripeRepository { return billingRepo.NewStripeRepository(string(apiKey), string(webhookSecret)) }
func ProvideCICDNamespace() CICDNamespace { return CICDNamespace("hexabase-cicd") }
func ProvideCICDProviderFactory(kubeClient kubernetes.Interface, k8sConfig *rest.Config, namespace CICDNamespace) cicdDomain.ProviderFactory { return cicdRepo.NewProviderFactory(kubeClient, k8sConfig, string(namespace)) }
func ProvideCICDCredentialManager(kubeClient kubernetes.Interface, namespace CICDNamespace) cicdDomain.CredentialManager { return cicdRepo.NewKubernetesCredentialManager(kubeClient, string(namespace)) }
func ProvideFunctionProviderFactory(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) functionDomain.ProviderFactory {
	return functionRepo.NewProviderFactory(kubeClient, dynamicClient)
}
func ProvideFunctionService(service *functionSvc.Service) functionDomain.Service {
	return service
}
func ProvideSQLDB(gormDB *gorm.DB) (*sql.DB, error) {
	return gormDB.DB()
}
func ProvideFunctionRepository(repo *functionRepo.PostgresRepository) functionDomain.Repository {
	return repo
}
func ProvideClickHouseConn(cfg *config.Config) (clickhouse.Conn, error) {
	// This should be expanded with full config options (user, pass, etc.)
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.ClickHouse.Address},
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func ProvideProxmoxRepository(cfg *config.Config) *nodeRepo.ProxmoxRepository {
	// TODO: Get Proxmox configuration from config
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	return nodeRepo.NewProxmoxRepository(httpClient, "https://proxmox.example.com/api2/json", "your-api-token")
}

func ProvideMetricsClientset(k8sConfig *rest.Config) (versioned.Interface, error) {
	return versioned.NewForConfig(k8sConfig)
}

func ProvideNodeService(svc *nodeService.Service) nodeDomain.Service {
	return svc
}

func ProvideNodeRepository(repo *nodeRepo.PostgresRepository) nodeDomain.Repository {
	return repo
}

func ProvideProxmoxRepositoryInterface(repo *nodeRepo.ProxmoxRepository) nodeDomain.ProxmoxRepository {
	return repo
}

func ProvideBackupProxmoxRepository(cfg *config.Config) backupDomain.ProxmoxRepository {
	// Reuse the same Proxmox connection settings
	// TODO: Get from config
	client := proxmoxRepo.NewClient("https://proxmox.example.com/api2/json", "root@pam", "tokenID", "tokenSecret")
	return backupRepo.NewProxmoxRepository(client)
}

func ProvideBackupEncryptionKey(cfg *config.Config) string {
	// TODO: Get encryption key from config - cfg.Backup not yet implemented
	return "your-backup-encryption-key"
}

func ProvideOllamaService(cfg *config.Config) aiopsDomain.LLMService {
	// TODO: Get Ollama configuration from config
	ollamaURL := "http://ollama.ollama.svc.cluster.local:11434"
	timeout := 30 * time.Second
	headers := make(map[string]string)
	return aiopsRepo.NewOllamaProvider(ollamaURL, timeout, headers)
}

func ProvideAIOpsProxyHandler(authSvc authDomain.Service, logger *slog.Logger, aiopsURL AIOpsServiceURL) (*aiopsHandler.AIOpsProxyHandler, error) {
	return aiopsHandler.NewAIOpsProxyHandler(authSvc, logger, string(aiopsURL))
}

// ProvideAIOpsAdapter provides an adapter from domain.Service to AIOpsService interface
func ProvideAIOpsAdapter(service aiopsDomain.Service) aiopsHandler.AIOpsService {
	return &aiopsServiceAdapter{service: service}
}

// aiopsServiceAdapter adapts domain.Service to AIOpsService interface for backward compatibility
type aiopsServiceAdapter struct {
	service aiopsDomain.Service
}

func (a *aiopsServiceAdapter) CreateChatSession(workspaceID, userID, model string) (*aiopsDomain.ChatSession, error) {
	ctx := context.Background()
	return a.service.CreateChatSession(ctx, workspaceID, userID, "", model)
}

func (a *aiopsServiceAdapter) GetChatSession(sessionID string) (*aiopsDomain.ChatSession, error) {
	ctx := context.Background()
	return a.service.GetChatSession(ctx, sessionID)
}

func (a *aiopsServiceAdapter) ListChatSessions(workspaceID string, limit, offset int) ([]*aiopsDomain.ChatSession, error) {
	ctx := context.Background()
	return a.service.ListChatSessions(ctx, workspaceID, limit, offset)
}

func (a *aiopsServiceAdapter) DeleteChatSession(sessionID string) error {
	ctx := context.Background()
	return a.service.DeleteChatSession(ctx, sessionID)
}

func (a *aiopsServiceAdapter) Chat(sessionID string, message string, contextInts []int) (*aiopsDomain.ChatResponse, error) {
	ctx := context.Background()
	chatMessage := aiopsDomain.ChatMessage{
		Role:    "user",
		Content: message,
	}
	return a.service.SendMessage(ctx, sessionID, chatMessage)
}

func (a *aiopsServiceAdapter) StreamChat(sessionID string, message string, contextInts []int) (<-chan *aiopsDomain.ChatStreamResponse, error) {
	ctx := context.Background()
	chatMessage := aiopsDomain.ChatMessage{
		Role:    "user",
		Content: message,
	}
	return a.service.StreamMessage(ctx, sessionID, chatMessage)
}

func (a *aiopsServiceAdapter) GetAvailableModels() ([]*aiopsDomain.ModelInfo, error) {
	ctx := context.Background()
	return a.service.ListAvailableModels(ctx)
}

func (a *aiopsServiceAdapter) GetTokenUsage(workspaceID, model string, limit, offset int) ([]*aiopsDomain.ModelUsage, error) {
	// This would need implementation based on the new service interface
	// For now, return empty slice
	return []*aiopsDomain.ModelUsage{}, nil
}

func ProvideInternalHandler(
	workspaceSvc workspaceDomain.Service,
	projectSvc projectDomain.Service,
	applicationSvc domain.Service,
	nodeSvc nodeDomain.Service,
	logSvc logsDomain.Service,
	monitoringSvc monitoringDomain.Service,
	aiopsSvc aiopsDomain.Service,
	cicdSvc cicdDomain.Service,
	backupSvc backupDomain.Service,
	logger *slog.Logger,
) *handlers.InternalHandler {
	return handlers.NewInternalHandler(
		workspaceSvc,
		projectSvc,
		applicationSvc,
		nodeSvc,
		logSvc,
		monitoringSvc,
		aiopsSvc,
		cicdSvc,
		backupSvc,
		logger,
	)
}

func InitializeApp(cfg *config.Config, db *gorm.DB, k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, k8sConfig *rest.Config, logger *slog.Logger) (*App, error) {
	wire.Build(
		ApplicationSet,
		AuthSet,
		BackupSet,
		BillingSet,
		MonitoringSet,
		NodeSet,
		OrganizationSet,
		ProjectSet,
		WorkspaceSet,
		CICDSet,
		FunctionSet,
		HelmSet,
		AIOpsSet,
		LogSet,
		InternalSet,
		ProvideOAuthProviderConfigs,
		ProvideStripeAPIKey,
		ProvideStripeWebhookSecret,
		ProvideCICDNamespace,
		ProvideMetricsClientset,
		NewApp,
	)
	return nil, nil
}

// ProvideTokenManager provides a TokenManager instance
func ProvideTokenManager(keyRepo authDomain.KeyRepository, cfg *config.Config) (*internalAuth.TokenManager, error) {
	privateKeyPEM, err := keyRepo.GetPrivateKey()
	if err != nil {
		return nil, err
	}

	publicKeyPEM, err := keyRepo.GetPublicKey()
	if err != nil {
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	return internalAuth.NewTokenManager(privateKey, publicKey, "https://api.hexabase-kaas.io", time.Hour), nil
}

// ProvideTokenDomainService provides a TokenDomainService instance
func ProvideTokenDomainService() authDomain.TokenDomainService {
	return authDomain.NewTokenDomainService()
}

// ProvideDefaultTokenExpiry provides the default token expiry
func ProvideDefaultTokenExpiry() int {
	return 3600 // 1 hour
}


// ProvideRedisClient provides a Redis client instance
func ProvideRedisClient(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	return redis.NewClient(&cfg.Redis, logger)
}