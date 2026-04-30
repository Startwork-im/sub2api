package service

func IsOpenAIPlatform(platform string) bool {
	return platform == PlatformOpenAI
}

func IsOpenAIChatPlatform(platform string) bool {
	return platform == PlatformOpenAIChat
}

func IsOpenAICompatiblePlatform(platform string) bool {
	return IsOpenAIPlatform(platform) || IsOpenAIChatPlatform(platform)
}
