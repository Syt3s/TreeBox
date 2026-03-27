package branding

// 定义常量
const (
	ProductName = "TreeBox"
	ServiceName = "treebox"
	BinaryName  = "TreeBox"

	ConfigPathEnvVar = "TREEBOX_CONFIG_PATH"

	LegacyConfigPathEnvVar = "NEKOBOX_CONFIG_PATH"

	AuthTokenCookieName = "treebox_token"

	LegacyAuthTokenCookieName = "nekobox_token"

	JWTIssuer = ProductName

	TelemetryNamespace = "treebox"

	GatewayHeaderFrom = "X-TreeBox-From"

	LegacyGatewayHeaderFrom   = "X-NekoBox-From"
	GatewayHeaderUserID       = "X-TreeBox-User-ID"
	LegacyGatewayHeaderUserID = "X-NekoBox-User-ID"
	GatewayName               = "treebox-gateway"
	LegacyGatewayName         = "nekobox-gateway"

	PixelUserHeader       = "treebox-user-id"
	LegacyPixelUserHeader = "neko-user-id"
)
