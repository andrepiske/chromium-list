package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

const (
	baseURL = "https://deb.debian.org/debian/pool/main/c/chromium/"
)

var (
	targetArch string
	targetDist string
)

type chromiumVersion struct {
	fullFilename string
	major        int
	minor        int
	build        int
	patch        int
	revision     int
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chromium-list",
		Short: "Find the latest Chromium version in Debian registry",
		Run:   run,
	}

	rootCmd.Flags().StringVar(&targetArch, "arch", "", "Target architecture (amd64, arm64, armhf, i386)")
	rootCmd.Flags().StringVar(&targetDist, "dist", "", "Target debian distribution version (e.g., 11, 12, 13, sid)")
	rootCmd.Flags().Bool("latest", true, "Show latest version (default)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	arch := targetArch
	if arch == "" {
		arch = detectArch()
	}

	dist := targetDist
	if dist == "" {
		dist = detectDist()
		if dist == "" {
			fmt.Println("Error: could not automatically detect debian distribution. Please provide --dist flag.")
			os.Exit(1)
		}
	}

	// Map known codenames to the expected filtering values
	if dist == "unstable" || dist == "trixie/sid" {
		dist = "sid"
	}

	resp, err := http.Get(baseURL)
	if err != nil {
		fmt.Printf("Error fetching URL: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	links, err := parseLinks(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing HTML: %v\n", err)
		os.Exit(1)
	}

	var versions []chromiumVersion
	distHasAnyPackages := false

	for _, link := range links {
		if !strings.HasPrefix(link, "chromium_") || !strings.HasSuffix(link, ".deb") {
			continue
		}

		// Filter by distribution
		matchesDist := false
		if dist == "sid" || dist == "testing" {
			// For sid, packages usually don't have ~deb in the revision, or have +b something.
			// e.g. chromium_146.0.7680.71-1_amd64.deb is sid.
			if !strings.Contains(link, "~deb") {
				matchesDist = true
			}
		} else {
			// We expect dist to be the numeric version, e.g. "12"
			debCode := fmt.Sprintf("~deb%s", dist)
			if strings.Contains(link, debCode) {
				matchesDist = true
			}
		}

		if !matchesDist {
			continue
		}

		distHasAnyPackages = true

		if !strings.HasSuffix(link, "_"+arch+".deb") {
			continue
		}

		v, err := parseVersion(link)
		if err == nil {
			versions = append(versions, v)
		}
	}

	if len(versions) == 0 {
		if !distHasAnyPackages {
			fmt.Printf("Distribution '%s' not found or invalid\n", dist)
		} else {
			fmt.Printf("No versions found for architecture: %s\n", arch)
		}
		os.Exit(1)
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j])
	})

	// Print latest (first one after sorting descending)
	fmt.Printf("%s%s\n", baseURL, versions[0].fullFilename)
}

func detectArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "386":
		return "i386"
	case "arm":
		return "armhf"
	default:
		return runtime.GOARCH
	}
}

func detectDist() string {
	// Try to read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VERSION_ID=") {
				return strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"'`)
			}
		}
	}
	
	// Fallback to /etc/debian_version to guess sid/testing
	data, err = os.ReadFile("/etc/debian_version")
	if err == nil {
		version := strings.TrimSpace(string(data))
		if version == "trixie/sid" || version == "testing" {
			// Without /etc/os-release we can't be sure if it's trixie or sid.
			// Let's assume sid as it's common for this case, or we could require explicit flag.
			// But returning "sid" works well for "trixie/sid" which is what's usually in testing.
			return "sid"
		}
		
		// If we couldn't find VERSION_ID, extract major number from debian_version
		if strings.Contains(version, ".") {
			return strings.Split(version, ".")[0]
		}
		if version != "" {
			return version
		}
	}
	
	return ""
}

func parseLinks(r io.Reader) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var links []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					links = append(links, a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return links, nil
}

func parseVersion(filename string) (chromiumVersion, error) {
	// Format: chromium_133.0.6943.53-1~deb12u1_amd64.deb
	// 1. Remove prefix "chromium_"
	// 2. Remove suffix "_<arch>.deb"
	// 3. What remains is "133.0.6943.53-1~deb12u1"

	// Better approach: split by underscore first, but filename might not have fixed underscores.
	// Actually, based on typical debian naming: <name>_<version>_<arch>.deb

	parts := strings.Split(filename, "_")
	if len(parts) < 3 {
		return chromiumVersion{}, fmt.Errorf("invalid filename format")
	}

	// version part is parts[1]
	// e.g. "133.0.6943.53-1~deb12u1"
	versionStr := parts[1]

	// Split by "-" to separate upstream version from debian revision
	// e.g. "133.0.6943.53" and "1~deb12u1"
	vParts := strings.Split(versionStr, "-")
	upstream := vParts[0]

	// Split upstream by "."
	nums := strings.Split(upstream, ".")
	if len(nums) != 4 {
		return chromiumVersion{}, fmt.Errorf("invalid version format")
	}

	major, _ := strconv.Atoi(nums[0])
	minor, _ := strconv.Atoi(nums[1])
	build, _ := strconv.Atoi(nums[2])
	patch, _ := strconv.Atoi(nums[3])

	// Parse revision strictly for sorting if needed, but usually upstream version is enough.
	// However, if upstream versions are identical, debian revision matters.
	// e.g. 1~deb12u1 vs 2~deb12u1
	// For simplicity, let's just parse the first digit of revision if present,
	// or ignore complex debian revision sorting for now as usually latest upstream is what we want.
	// But let's try to get a simple integer from the start of revision if possible.
	rev := 0
	if len(vParts) > 1 {
		// "1~deb12u1" -> "1"
		revStr := strings.Split(vParts[1], "~")[0]
		// It might be just "1" or "1+b1" etc.
		// Just take leading digits
		var digits string
		for _, r := range revStr {
			if r >= '0' && r <= '9' {
				digits += string(r)
			} else {
				break
			}
		}
		if digits != "" {
			rev, _ = strconv.Atoi(digits)
		}
	}

	return chromiumVersion{
		fullFilename: filename,
		major:        major,
		minor:        minor,
		build:        build,
		patch:        patch,
		revision:     rev,
	}, nil
}

func compareVersions(v1, v2 chromiumVersion) bool {
	if v1.major != v2.major {
		return v1.major > v2.major
	}
	if v1.minor != v2.minor {
		return v1.minor > v2.minor
	}
	if v1.build != v2.build {
		return v1.build > v2.build
	}
	if v1.patch != v2.patch {
		return v1.patch > v2.patch
	}
	return v1.revision > v2.revision
}
