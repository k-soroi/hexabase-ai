package routes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authHandler "github.com/hexabase/hexabase-ai/api/internal/auth/handler"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/server/ogen"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/wire"
	"github.com/hexabase/hexabase-ai/api/internal/shared/utils/httpauth"
)

// requireInternalAuth is middleware that validates internal service authentication
func requireInternalAuth(authHandler *authHandler.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for internal service token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		if !httpauth.HasBearerPrefix(authHeader) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := httpauth.TrimBearerPrefix(authHeader)

		// Validate internal service token
		// This should validate against service account tokens or special AI agent tokens
		if !isValidInternalToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid internal service token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isValidInternalToken validates internal service tokens
func isValidInternalToken(token string) bool {
	// TODO: Implement proper validation against service account tokens
	// For now, check for a specific prefix or validate against a service registry
	return strings.HasPrefix(token, "internal_") || strings.HasPrefix(token, "ai_agent_")
}

// wrapOgenHandler wraps an ogen server as a Gin handler
func wrapOgenHandler(ogenServer *ogen.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set default Content-Type if not provided (for compatibility with existing behavior)
		if c.Request.Header.Get("Content-Type") == "" && c.Request.Method == http.MethodPost {
			c.Request.Header.Set("Content-Type", "application/json")
		}

		ogenServer.ServeHTTP(c.Writer, c.Request)
	}
}

// SetupRoutes configures all API routes
//nolint:funlen,maintidx // TODO: Refactor this function to reduce complexity
func SetupRoutes(router *gin.Engine, app *wire.App) error {
	// API version 1
	v1 := router.Group("/api/v1")

	// Create ogen server for OpenAPI-based endpoints
	ogenServer, err := ogen.NewServer(app.OgenAuthHandler)
	if err != nil {
		return fmt.Errorf("failed to create ogen server: %w", err)
	}

	// Authentication routes
	auth := router.Group("/auth")
	{
		auth.POST("/login/:provider", app.AuthHandler.Login)
		auth.POST("/sign-up/:provider", wrapOgenHandler(ogenServer))
		auth.GET("/callback/:provider", app.AuthHandler.Callback)
		auth.POST("/callback/:provider", app.AuthHandler.Callback) // For PKCE flow
		auth.POST("/refresh", app.AuthHandler.RefreshToken)
		auth.POST("/logout", app.AuthHandler.AuthMiddleware(), app.AuthHandler.Logout)
		auth.GET("/me", app.AuthHandler.AuthMiddleware(), app.AuthHandler.GetCurrentUser)

		// Session management
		auth.GET("/sessions", app.AuthHandler.AuthMiddleware(), app.AuthHandler.GetSessions)
		auth.DELETE("/sessions/:sessionId", app.AuthHandler.AuthMiddleware(), app.AuthHandler.RevokeSession)
		auth.POST("/sessions/revoke-all", app.AuthHandler.AuthMiddleware(), app.AuthHandler.RevokeAllSessions)

		// Security logs
		auth.GET("/security-logs", app.AuthHandler.AuthMiddleware(), app.AuthHandler.GetSecurityLogs)
	}

	// JWKS endpoint (public)
	router.GET("/.well-known/jwks.json", app.AuthHandler.JWKS)

	// Protected routes (require authentication)
	protected := v1.Group("")
	protected.Use(app.AuthHandler.AuthMiddleware())

	// AIOps Chat Proxy
	ai := protected.Group("/ai")
	{
		// This single endpoint proxies all chat-related interactions to the Python service.
		// The python service is responsible for handling sessions, history, etc.
		// The `workspace_id` is passed as a query param to give context to the AIOps service.
		ai.Any("/chat", app.AIOpsProxyHandler.ChatProxy)
	}

	// Organization routes
	orgs := protected.Group("/organizations")
	{
		orgs.POST("/", app.OrganizationHandler.CreateOrganization)
		orgs.GET("/", app.OrganizationHandler.ListOrganizations)
		orgs.GET("/:orgId", app.OrganizationHandler.GetOrganization)
		orgs.PATCH("/:orgId", app.OrganizationHandler.UpdateOrganization)
		orgs.DELETE("/:orgId", app.OrganizationHandler.DeleteOrganization)

		// Organization members
		// orgs.POST("/:orgId/members", app.OrganizationHandler.AddMember) // Removed - use invitations instead
		orgs.GET("/:orgId/members", app.OrganizationHandler.ListMembers)
		orgs.DELETE("/:orgId/members/:userId", app.OrganizationHandler.RemoveMember)
		orgs.PUT("/:orgId/members/:userId/role", app.OrganizationHandler.UpdateMemberRole)

		// Organization invitations
		orgs.POST("/:orgId/invitations", app.OrganizationHandler.InviteUser)
		orgs.GET("/:orgId/invitations", app.OrganizationHandler.ListPendingInvitations)
		orgs.POST("/invitations/:token/accept", app.OrganizationHandler.AcceptInvitation)
		orgs.DELETE("/invitations/:invitationId", app.OrganizationHandler.CancelInvitation)

		// Organization activity
		orgs.GET("/:orgId/stats", app.OrganizationHandler.GetOrganizationStats)

		// Billing under organizations
		billing := orgs.Group("/:orgId/billing")
		{
			billing.GET("/subscription", app.BillingHandler.GetSubscription)
			billing.POST("/subscription", app.BillingHandler.CreateSubscription)
			billing.PUT("/subscription", app.BillingHandler.UpdateSubscription)
			billing.DELETE("/subscription", app.BillingHandler.CancelSubscription)

			billing.GET("/payment-methods", app.BillingHandler.ListPaymentMethods)
			billing.POST("/payment-methods", app.BillingHandler.AddPaymentMethod)
			billing.PUT("/payment-methods/:methodId/default", app.BillingHandler.SetDefaultPaymentMethod)
			billing.DELETE("/payment-methods/:methodId", app.BillingHandler.RemovePaymentMethod)

			billing.GET("/invoices", app.BillingHandler.ListInvoices)
			billing.GET("/invoices/upcoming", app.BillingHandler.GetUpcomingInvoice)
			billing.GET("/invoices/:invoiceId", app.BillingHandler.GetInvoice)
			billing.GET("/invoices/:invoiceId/download", app.BillingHandler.DownloadInvoice)

			billing.GET("/usage/current", app.BillingHandler.GetCurrentUsage)
			billing.GET("/overview", app.BillingHandler.GetBillingOverview)
			billing.GET("/settings", app.BillingHandler.GetBillingSettings)
			billing.PUT("/settings", app.BillingHandler.UpdateBillingSettings)
		}

		// Workspaces under organizations
		workspaces := orgs.Group("/:orgId/workspaces")
		{
			workspaces.POST("/", app.WorkspaceHandler.CreateWorkspace)
			workspaces.GET("/", app.WorkspaceHandler.ListWorkspaces)
			workspaces.GET("/:wsId", app.WorkspaceHandler.GetWorkspace)
			workspaces.PUT("/:wsId", app.WorkspaceHandler.UpdateWorkspace)
			workspaces.DELETE("/:wsId", app.WorkspaceHandler.DeleteWorkspace)
			workspaces.GET("/:wsId/kubeconfig", app.WorkspaceHandler.GetKubeconfig)
			workspaces.GET("/:wsId/resource-usage", app.WorkspaceHandler.GetResourceUsage)

			// Workspace members
			workspaces.POST("/:wsId/members", app.WorkspaceHandler.AddWorkspaceMember)
			workspaces.GET("/:wsId/members", app.WorkspaceHandler.ListWorkspaceMembers)
			workspaces.DELETE("/:wsId/members/:userId", app.WorkspaceHandler.RemoveWorkspaceMember)
		}
	}

	// Workspace-scoped routes
	workspaceScoped := protected.Group("/workspaces/:wsId")
	{
		// Projects
		projects := workspaceScoped.Group("/projects")
		{
			projects.POST("/", app.ProjectHandler.CreateProject)
			projects.GET("/", app.ProjectHandler.ListProjects)
			projects.GET("/:projectId", app.ProjectHandler.GetProject)
			projects.PUT("/:projectId", app.ProjectHandler.UpdateProject)
			projects.DELETE("/:projectId", app.ProjectHandler.DeleteProject)

			// Sub-projects
			projects.POST("/:projectId/subprojects", app.ProjectHandler.CreateSubProject)
			projects.GET("/:projectId/hierarchy", app.ProjectHandler.GetProjectHierarchy)

			// Resource management
			projects.POST("/:projectId/resource-quota", app.ProjectHandler.ApplyResourceQuota)
			projects.GET("/:projectId/resource-usage", app.ProjectHandler.GetResourceUsage)

			// Project members
			projects.POST("/:projectId/members", app.ProjectHandler.AddProjectMember)
			projects.GET("/:projectId/members", app.ProjectHandler.ListProjectMembers)
			projects.DELETE("/:projectId/members/:userId", app.ProjectHandler.RemoveProjectMember)

			// Activity logs
			projects.GET("/:projectId/activity", app.ProjectHandler.GetActivityLogs)
		}

		// AIOps (workspace level)
		aiops := workspaceScoped.Group("/aiops")
		{
			// For now, we only have the chat proxy endpoint
			// The Python service will handle all the session management and other features
			aiops.Any("/chat", app.AIOpsProxyHandler.ChatProxy)
		}

		// Monitoring (workspace level)
		monitoring := workspaceScoped.Group("/monitoring")
		{
			monitoring.GET("/metrics", app.MonitoringHandler.GetMetrics)
			monitoring.GET("/health", app.MonitoringHandler.GetClusterHealth)
			monitoring.GET("/alerts", app.MonitoringHandler.GetAlerts)
			monitoring.POST("/alerts", app.MonitoringHandler.CreateAlert)
			monitoring.PUT("/alerts/:alertId/acknowledge", app.MonitoringHandler.AcknowledgeAlert)
			monitoring.PUT("/alerts/:alertId/resolve", app.MonitoringHandler.ResolveAlert)
		}

		// Nodes (workspace level)
		nodes := workspaceScoped.Group("/nodes")
		{
			nodes.POST("/", app.NodeHandler.ProvisionDedicatedNode)
			nodes.GET("/", app.NodeHandler.ListNodes)
			nodes.GET("/:nodeId", app.NodeHandler.GetNode)
			nodes.DELETE("/:nodeId", app.NodeHandler.DeleteNode)

			// Node operations
			nodes.POST("/:nodeId/start", app.NodeHandler.StartNode)
			nodes.POST("/:nodeId/stop", app.NodeHandler.StopNode)
			nodes.POST("/:nodeId/reboot", app.NodeHandler.RebootNode)

			// Node monitoring
			nodes.GET("/:nodeId/status", app.NodeHandler.GetNodeStatus)
			nodes.GET("/:nodeId/metrics", app.NodeHandler.GetNodeMetrics)
			nodes.GET("/:nodeId/events", app.NodeHandler.GetNodeEvents)

			// Resource usage
			nodes.GET("/usage", app.NodeHandler.GetWorkspaceResourceUsage)
			nodes.GET("/costs", app.NodeHandler.GetNodeCosts)
			nodes.POST("/check-allocation", app.NodeHandler.CanAllocateResources)

			// Plan transitions
			nodes.POST("/transition/shared", app.NodeHandler.TransitionToSharedPlan)
			nodes.POST("/transition/dedicated", app.NodeHandler.TransitionToDedicatedPlan)
		}

		// Applications (workspace level)
		applications := workspaceScoped.Group("/applications")
		{
			applications.POST("/", app.ApplicationHandler.CreateApplication)
			applications.GET("/", app.ApplicationHandler.ListApplications)
			applications.GET("/:appId", app.ApplicationHandler.GetApplication)
			applications.PUT("/:appId", app.ApplicationHandler.UpdateApplication)
			applications.DELETE("/:appId", app.ApplicationHandler.DeleteApplication)

			// Application operations
			applications.POST("/:appId/start", app.ApplicationHandler.StartApplication)
			applications.POST("/:appId/stop", app.ApplicationHandler.StopApplication)
			applications.POST("/:appId/restart", app.ApplicationHandler.RestartApplication)
			applications.POST("/:appId/scale", app.ApplicationHandler.ScaleApplication)

			// Pod operations
			applications.GET("/:appId/pods", app.ApplicationHandler.ListPods)
			applications.POST("/:appId/pods/:podName/restart", app.ApplicationHandler.RestartPod)
			applications.GET("/:appId/logs", app.ApplicationHandler.GetPodLogs)
			applications.GET("/:appId/logs/stream", app.ApplicationHandler.StreamPodLogs)

			// Monitoring
			applications.GET("/:appId/metrics", app.ApplicationHandler.GetApplicationMetrics)
			applications.GET("/:appId/events", app.ApplicationHandler.GetApplicationEvents)

			// Network operations
			applications.PUT("/:appId/network", app.ApplicationHandler.UpdateNetworkConfig)
			applications.GET("/:appId/endpoints", app.ApplicationHandler.GetApplicationEndpoints)

			// Node operations
			applications.PUT("/:appId/node-affinity", app.ApplicationHandler.UpdateNodeAffinity)
			applications.POST("/:appId/migrate", app.ApplicationHandler.MigrateToNode)

			// CronJob operations
			applications.PUT("/:appId/schedule", app.ApplicationHandler.UpdateCronJobSchedule)
			applications.POST("/:appId/trigger", app.ApplicationHandler.TriggerCronJob)
			applications.GET("/:appId/executions", app.ApplicationHandler.GetCronJobExecutions)
			applications.GET("/:appId/cronjob-status", app.ApplicationHandler.GetCronJobStatus)

			// Function operations
			applications.POST("/:appId/versions", app.ApplicationHandler.DeployFunctionVersion)
			applications.GET("/:appId/versions", app.ApplicationHandler.GetFunctionVersions)
			applications.PUT("/:appId/versions/:versionId/active", app.ApplicationHandler.SetActiveFunctionVersion)
			applications.POST("/:appId/invoke", app.ApplicationHandler.InvokeFunction)
			applications.GET("/:appId/invocations", app.ApplicationHandler.GetFunctionInvocations)
			applications.GET("/:appId/function-events", app.ApplicationHandler.GetFunctionEvents)

			// Backup policies (application level - dedicated plan only)
			applications.POST("/:appId/backup-policy", app.BackupHandler.CreateBackupPolicy)
			applications.GET("/:appId/backup-policy", app.BackupHandler.GetBackupPolicy)
			applications.PUT("/:appId/backup-policy", app.BackupHandler.UpdateBackupPolicy)
			applications.DELETE("/:appId/backup-policy", app.BackupHandler.DeleteBackupPolicy)

			// Backup operations (application level)
			applications.POST("/:appId/backups/trigger", app.BackupHandler.TriggerManualBackup)
			applications.GET("/:appId/backups", app.BackupHandler.ListBackupExecutions)
			applications.GET("/:appId/backups/latest", app.BackupHandler.GetLatestBackup)
			applications.GET("/:appId/backups/:backupId", app.BackupHandler.GetBackupExecution)
			applications.GET("/:appId/backups/:backupId/manifest", app.BackupHandler.GetBackupManifest)
			applications.GET("/:appId/backups/:backupId/download", app.BackupHandler.DownloadBackup)
			applications.POST("/:appId/backups/:backupId/restore", app.BackupHandler.RestoreBackup)

			// Restore operations (application level)
			applications.GET("/:appId/restores", app.BackupHandler.ListBackupRestores)
			applications.GET("/:appId/restores/:restoreId", app.BackupHandler.GetBackupRestore)
		}

		// Function management (workspace level)
		functions := workspaceScoped.Group("/functions")
		{
			// Function CRUD
			functions.POST("/", app.FunctionHandler.CreateFunctionGin)
			functions.GET("/", app.FunctionHandler.ListFunctionsGin)
			functions.GET("/:functionId", app.FunctionHandler.GetFunctionGin)
			functions.PUT("/:functionId", app.FunctionHandler.UpdateFunctionGin)
			functions.DELETE("/:functionId", app.FunctionHandler.DeleteFunctionGin)

			// Version management
			functions.POST("/:functionId/versions", app.FunctionHandler.DeployVersionGin)
			functions.GET("/:functionId/versions", app.FunctionHandler.ListVersionsGin)
			functions.GET("/:functionId/versions/:versionId", app.FunctionHandler.GetVersionGin)
			functions.PUT("/:functionId/versions/:versionId/active", app.FunctionHandler.SetActiveVersionGin)
			functions.POST("/:functionId/rollback", app.FunctionHandler.RollbackVersionGin)

			// Trigger management
			functions.POST("/:functionId/triggers", app.FunctionHandler.CreateTriggerGin)
			functions.GET("/:functionId/triggers", app.FunctionHandler.ListTriggersGin)
			functions.PUT("/:functionId/triggers/:triggerId", app.FunctionHandler.UpdateTriggerGin)
			functions.DELETE("/:functionId/triggers/:triggerId", app.FunctionHandler.DeleteTriggerGin)

			// Invocation
			functions.POST("/:functionId/invoke", app.FunctionHandler.InvokeFunctionGin)
			functions.POST("/:functionId/invoke-async", app.FunctionHandler.InvokeFunctionAsyncGin)
			functions.GET("/invocations/:invocationId", app.FunctionHandler.GetInvocationStatusGin)
			functions.GET("/:functionId/invocations", app.FunctionHandler.ListInvocationsGin)

			// Monitoring
			functions.GET("/:functionId/logs", app.FunctionHandler.GetFunctionLogsGin)
			functions.GET("/:functionId/metrics", app.FunctionHandler.GetFunctionMetricsGin)
			functions.GET("/:functionId/events", app.FunctionHandler.GetFunctionEventsGin)

			// Provider capabilities
			functions.GET("/provider/capabilities", app.FunctionHandler.GetProviderCapabilitiesGin)
			functions.GET("/provider/health", app.FunctionHandler.GetProviderHealthGin)
		}

		// Function event processing (not scoped to a specific function)
		events := workspaceScoped.Group("/function-events")
		{
			events.POST("/:eventId/process", app.ApplicationHandler.ProcessFunctionEvent)
		}

		// CI/CD (workspace level)
		pipelines := workspaceScoped.Group("/pipelines")
		{
			pipelines.POST("/", app.CICDHandler.CreatePipeline)
			pipelines.GET("/", app.CICDHandler.ListPipelines)
			pipelines.POST("/from-template", app.CICDHandler.CreatePipelineFromTemplate)
		}

		// Credentials (workspace level)
		credentials := workspaceScoped.Group("/credentials")
		{
			credentials.POST("/git", app.CICDHandler.CreateGitCredential)
			credentials.POST("/registry", app.CICDHandler.CreateRegistryCredential)
			credentials.GET("/", app.CICDHandler.ListCredentials)
			credentials.DELETE("/:credentialName", app.CICDHandler.DeleteCredential)
		}

		// Provider config (workspace level)
		workspaceScoped.GET("/provider-config", app.CICDHandler.GetProviderConfig)
		workspaceScoped.PUT("/provider-config", app.CICDHandler.SetProviderConfig)

		// Backup storage (workspace level - dedicated plan only)
		backupStorages := workspaceScoped.Group("/backup-storages")
		{
			backupStorages.POST("/", app.BackupHandler.CreateBackupStorage)
			backupStorages.GET("/", app.BackupHandler.ListBackupStorages)
			backupStorages.GET("/:storageId", app.BackupHandler.GetBackupStorage)
			backupStorages.PUT("/:storageId", app.BackupHandler.UpdateBackupStorage)
			backupStorages.DELETE("/:storageId", app.BackupHandler.DeleteBackupStorage)
			backupStorages.GET("/:storageId/usage", app.BackupHandler.GetStorageUsage)
		}

		// Storage usage (workspace level)
		workspaceScoped.GET("/backup-storage-usage", app.BackupHandler.GetWorkspaceStorageUsage)
	}

	// Pipeline-specific routes (not workspace-scoped)
	pipelineRoutes := protected.Group("/pipelines")
	{
		pipelineRoutes.GET("/:pipelineId", app.CICDHandler.GetPipeline)
		pipelineRoutes.DELETE("/:pipelineId", app.CICDHandler.DeletePipeline)
		pipelineRoutes.POST("/:pipelineId/cancel", app.CICDHandler.CancelPipeline)
		pipelineRoutes.POST("/:pipelineId/retry", app.CICDHandler.RetryPipeline)
		pipelineRoutes.GET("/:pipelineId/logs", app.CICDHandler.GetPipelineLogs)
		pipelineRoutes.GET("/:pipelineId/logs/stream", app.CICDHandler.StreamPipelineLogs)

		// Templates
		pipelineRoutes.GET("/templates", app.CICDHandler.ListTemplates)
		pipelineRoutes.GET("/templates/:templateId", app.CICDHandler.GetTemplate)
	}

	// CI/CD Providers (global)
	protected.GET("/providers", app.CICDHandler.ListProviders)

	// Billing plans (public)
	plans := v1.Group("/plans")
	{
		plans.GET("/", app.BillingHandler.ListPlans)
		plans.GET("/compare", app.BillingHandler.ComparePlans)
	}

	// Node plans (public)
	nodePlans := v1.Group("/node-plans")
	{
		nodePlans.GET("/", app.NodeHandler.GetAvailablePlans)
		nodePlans.GET("/:planId", app.NodeHandler.GetPlanDetails)
	}

	// Webhook routes (no authentication required)
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/stripe", app.BillingHandler.HandleStripeWebhook)
	}

	// Internal API routes (for AI agents and internal services)
	// These routes require special authentication (e.g., service account tokens)
	internal := router.Group("/internal/v1")
	internal.Use(requireInternalAuth(app.AuthHandler))
	{
		// System-wide operations
		internal.GET("/health", app.InternalHandler.GetSystemHealth)

		// Workspace operations
		workspaceInternal := internal.Group("/workspaces/:workspaceId")
		{
			// Overview and insights
			workspaceInternal.GET("/overview", app.InternalHandler.GetWorkspaceOverview)
			workspaceInternal.POST("/insights", app.InternalHandler.GetAIInsights)

			// Operations
			workspaceInternal.POST("/operations", app.InternalHandler.ExecuteWorkspaceOperation)
			workspaceInternal.GET("/nodes", app.InternalHandler.GetNodes)
			workspaceInternal.POST("/deployments/:deploymentName/scale", app.InternalHandler.ScaleDeployment)
		}

		// Application operations
		applicationInternal := internal.Group("/applications/:appId")
		{
			applicationInternal.GET("/details", app.InternalHandler.GetApplicationDetails)
			applicationInternal.POST("/autoscale", app.InternalHandler.AutoScaleApplication)
			applicationInternal.POST("/backup", app.InternalHandler.TriggerBackup)
			applicationInternal.GET("/performance", app.InternalHandler.AnalyzePerformance)
		}

		// Incident management
		internal.POST("/incidents/:incidentId", app.InternalHandler.ManageIncident)

		// Log queries
		internal.POST("/logs/query", app.InternalHandler.QueryLogs)
	}

	// Catch-all for undefined routes
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint not found",
			"path":  c.Request.URL.Path,
		})
	})

	return nil
}
