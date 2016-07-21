package analyzer

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// ProjectDependenciesModel ...
type ProjectDependenciesModel struct {
	ComplieSDKVersion *version.Version
	BuildToolsVersion *version.Version

	UseSupportLibrary     bool
	UseGooglePlayServices bool
}

// NewProjectDependenciesModel ...
func NewProjectDependenciesModel() *ProjectDependenciesModel {
	return &ProjectDependenciesModel{}
}

// SetComplieSDKVersion ...
func (dependencies *ProjectDependenciesModel) SetComplieSDKVersion(complieSDKVersionStr string) error {
	complieSDKVersion, err := version.NewVersion(complieSDKVersionStr)
	if err != nil {
		return err
	}

	dependencies.ComplieSDKVersion = complieSDKVersion
	return nil
}

// SetBuildToolsVersion ...
func (dependencies *ProjectDependenciesModel) SetBuildToolsVersion(buildToolsVersionStr string) error {
	buildToolsVersion, err := version.NewVersion(buildToolsVersionStr)
	if err != nil {
		return err
	}

	dependencies.BuildToolsVersion = buildToolsVersion
	return nil
}

// ParseComplieSDKVersion ...
func (dependencies *ProjectDependenciesModel) ParseComplieSDKVersion(buildGradleContent, gradlePropertiesContent string) error {
	compileSDKVersionStr, err := parseCompileSDKVersion(buildGradleContent)
	if err != nil {
		return fmt.Errorf("Failed to parse compile sdk version from build.gradle, error: %s", err)
	}

	compileSDKVesrion, err := version.NewVersion(compileSDKVersionStr)
	if err != nil {
		// Possible defined with variable

		// Search for var in build.gradle
		compileSDKVersionStr, err = findVariable(buildGradleContent, compileSDKVersionStr)
		if err != nil {
			return fmt.Errorf("Failed to find variable (%s) in build.gradle, error: %s", compileSDKVersionStr, err)
		}

		compileSDKVesrion, err = version.NewVersion(compileSDKVersionStr)
		if err != nil {
			// Search for var in gradle.properties
			compileSDKVersionStr, err = findVariable(gradlePropertiesContent, compileSDKVersionStr)
			if err != nil {
				return fmt.Errorf("Failed to find variable (%s) in gradle.properties, error: %s", compileSDKVersionStr, err)
			}
			compileSDKVesrion, err = version.NewVersion(compileSDKVersionStr)
			if err != nil {
				return fmt.Errorf("Failed to parse (%s), error: %s", compileSDKVersionStr, err)
			}
		}
	}

	if compileSDKVesrion == nil {
		return fmt.Errorf("Failed to parse (%s)", compileSDKVersionStr)
	}

	dependencies.ComplieSDKVersion = compileSDKVesrion
	return nil
}

// ParseBuildToolsVersion ...
func (dependencies *ProjectDependenciesModel) ParseBuildToolsVersion(buildGradleContent, gradlePropertiesContent string) error {
	buildToolsVersionStr, err := parseBuildToolsVersion(buildGradleContent)
	if err != nil {
		return fmt.Errorf("Failed to parse build tools version from build.gradle, error: %s", err)
	}

	buildToolsVersion, err := version.NewVersion(buildToolsVersionStr)
	if err != nil {
		// Possible defined with variable

		// Search for var in build.gradle
		buildToolsVersionStr, err = findVariable(buildGradleContent, buildToolsVersionStr)
		if err != nil {
			return fmt.Errorf("Failed to find variable (%s) in build.gradle, error: %s", buildToolsVersionStr, err)
		}

		buildToolsVersion, err = version.NewVersion(buildToolsVersionStr)
		if err != nil {
			// Search for var in gradle.properties
			buildToolsVersionStr, err = findVariable(gradlePropertiesContent, buildToolsVersionStr)
			if err != nil {
				return fmt.Errorf("Failed to find variable (%s) in gradle.properties, error: %s", buildToolsVersionStr, err)
			}
			buildToolsVersion, err = version.NewVersion(buildToolsVersionStr)
			if err != nil {
				return fmt.Errorf("Failed to parse (%s), error: %s", buildToolsVersionStr, err)
			}
		}
	}

	if buildToolsVersion == nil {
		return fmt.Errorf("Failed to parse (%s)", buildToolsVersionStr)
	}

	dependencies.BuildToolsVersion = buildToolsVersion
	return nil
}

// ParseUseSupportLibrary ...
func (dependencies *ProjectDependenciesModel) ParseUseSupportLibrary(buildGradleContent string) error {
	useSupportLibrary, err := parseUseSupportLibrary(buildGradleContent)
	if err != nil {
		return fmt.Errorf("Failed to parse support library usage, error: %s", err)
	}

	dependencies.UseSupportLibrary = useSupportLibrary
	return nil
}

// ParseUseGooglePlayServices ...
func (dependencies *ProjectDependenciesModel) ParseUseGooglePlayServices(buildGradleContent string) error {
	useGooglePlayServices, err := parseUseGooglePlayServices(buildGradleContent)
	if err != nil {
		return fmt.Errorf("Failed to parse google play service usage, error: %s", err)
	}

	dependencies.UseGooglePlayServices = useGooglePlayServices
	return nil
}

// String ...
func (dependencies ProjectDependenciesModel) String() string {
	outStr := ""
	if dependencies.ComplieSDKVersion != nil {
		outStr += fmt.Sprintf("  compileSdkVersion: %s\n", dependencies.ComplieSDKVersion.String())
	}
	if dependencies.BuildToolsVersion != nil {
		outStr += fmt.Sprintf("  buildToolsVersion: %s\n", dependencies.BuildToolsVersion.String())
	}

	outStr += fmt.Sprintf("  uses Support Library: %v\n", dependencies.UseSupportLibrary)
	outStr += fmt.Sprintf("  uses Google Play Services: %v\n", dependencies.UseGooglePlayServices)
	return outStr
}

// ParseIncludedModules ...
func ParseIncludedModules(settingsGradleContent string) ([]string, error) {
	// include ':app', ':dynamicgrid'
	includeRegexp := regexp.MustCompile(`\s*include\s*(?P<modules>.*)`)
	modules := []string{}

	scanner := bufio.NewScanner(strings.NewReader(settingsGradleContent))
	for scanner.Scan() {
		matches := includeRegexp.FindStringSubmatch(scanner.Text())

		if len(matches) > 1 {
			includeStr := matches[1]
			splits := strings.Split(includeStr, ",")
			for _, split := range splits {
				module := strings.TrimSpace(split)

				if strings.HasPrefix(module, `'`) {
					module = strings.Trim(module, "'")
				} else if strings.HasPrefix(module, `"`) {
					module = strings.Trim(module, `"`)
				}

				if strings.HasPrefix(module, ":") {
					module = strings.TrimPrefix(module, ":")
				}

				modules = append(modules, module)
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	return modules, nil
}

// -----------------------
// --- Functions
// -----------------------

func parseCompileSDKVersion(buildGradleContent string) (string, error) {
	//     compileSdkVersion 23
	compileSDKVersionRegexp := regexp.MustCompile(`\s*compileSdkVersion (?P<version>.+)`)
	compileSDKVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := compileSDKVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 1 {
			compileSDKVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if compileSDKVersionStr == "" {
		return "", errors.New("Failed to find compileSdkVersion")
	}

	compileSDKVersionStr = strings.Trim(compileSDKVersionStr, `"`)
	compileSDKVersionStr = strings.Trim(compileSDKVersionStr, `'`)

	return compileSDKVersionStr, nil
}

func parseBuildToolsVersion(buildGradleContent string) (string, error) {
	//     buildToolsVersion "23.0.3"
	buildToolsVersionRegexp := regexp.MustCompile(`\s*buildToolsVersion (?P<version>.+)`)
	buildToolsVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := buildToolsVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 1 {
			buildToolsVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if buildToolsVersionStr == "" {
		return "", errors.New("Failed to find buildToolsVersion")
	}

	buildToolsVersionStr = strings.Trim(buildToolsVersionStr, `"`)
	buildToolsVersionStr = strings.Trim(buildToolsVersionStr, `'`)

	return buildToolsVersionStr, nil
}

func parseUseSupportLibrary(buildGradleContent string) (bool, error) {
	//     compile "com.android.support:appcompat-v7:23.4.0"
	//     compile "com.android.support:23.4.0"
	supportLibraryVersionRegexp := regexp.MustCompile(`\s*compile\s*\"com.android.support:(?P<tool>.[^:]*):*(?P<version>.*)\"`)
	supportLibraryVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := supportLibraryVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 2 {
			supportLibraryVersionStr = matches[2]
			break
		} else if len(matches) > 1 {
			supportLibraryVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return (supportLibraryVersionStr != ""), nil
}

func parseUseGooglePlayServices(buildGradleContent string) (bool, error) {
	//     compile "com.google.android.gms:play-services-location:7.8.0"
	//     compile "com.google.android.gms:7.8.0"
	googlePlayServicesVersionRegexp := regexp.MustCompile(`\s*compile\s*\"com.google.android.gms:(?P<tool>.[^:]*):*(?P<version>.*)\"`)
	googlePlayServicesVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := googlePlayServicesVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 2 {
			googlePlayServicesVersionStr = matches[2]
			break
		} else if len(matches) > 1 {
			googlePlayServicesVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return (googlePlayServicesVersionStr != ""), nil
}

func findVariable(content, variable string) (string, error) {
	value := ""

	variableExpStr := `.*` + variable + `\s*=\s*["']*(?P<value>.*)["']*`
	variableExp := regexp.MustCompile(variableExpStr)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		matches := variableExp.FindStringSubmatch(scanner.Text())
		if len(matches) > 1 {
			value = matches[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return value, nil
}
