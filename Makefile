.PHONY: check-dep
check-dep:
	cd ./methanol/client && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./methanol/server && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./json-b && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./json-view && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./mqtt/subscriber && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./angular-i18n-demo/built-in && ncu
	cd ./angular-i18n-demo/transloco && ncu
	cd ./x509client/client && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	cd ./x509client/server && ./mvnw.cmd versions:display-dependency-updates && ./mvnw.cmd versions:display-plugin-updates
	