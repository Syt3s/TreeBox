package branding

const (
	ProductName = "TreeBox"
	ServiceName = "treebox"
	BinaryName  = "TreeBox"

	ConfigPathEnvVar = "TREEBOX_CONFIG_PATH"
	// LegacyConfigPathEnvVar keeps old deployments working during the rename.
	LegacyConfigPathEnvVar = "NEKOBOX_CONFIG_PATH"

	AuthTokenCookieName = "treebox_token"
	// LegacyAuthTokenCookieName keeps existing browser sessions valid.
	LegacyAuthTokenCookieName = "nekobox_token"

	JWTIssuer = ProductName

	TelemetryNamespace = "treebox"

	GatewayHeaderFrom = "X-TreeBox-From"
	// Legacy gateway headers are still sent for downstream compatibility.
	LegacyGatewayHeaderFrom   = "X-NekoBox-From"
	GatewayHeaderUserID       = "X-TreeBox-User-ID"
	LegacyGatewayHeaderUserID = "X-NekoBox-User-ID"
	GatewayName               = "treebox-gateway"
	LegacyGatewayName         = "nekobox-gateway"

	PixelUserHeader       = "treebox-user-id"
	LegacyPixelUserHeader = "neko-user-id"
)
