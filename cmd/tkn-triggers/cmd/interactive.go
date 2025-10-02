/*
Copyright 2025 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func runInteractiveSetup() error {
	reader := bufio.NewReader(os.Stdin)
	if githubRepo != "" && publicDomain != "" && githubToken != "" {
		log.Println("GitHub configuration already there")
		return nil
	}
	// Get GitHub repository
	if githubRepo == "" {
		log.Print("\nEnter your GitHub repository (owner/repo): ")
		repo, _ := reader.ReadString('\n')
		githubRepo = strings.TrimSpace(repo)
		if githubRepo == "" {
			log.Println("Skipping GitHub integration.")
			return nil
		}
	}

	// Get public domain
	if publicDomain == "" {
		log.Print("Enter your public route URL (e.g. myapp.example.com): ")
		domain, _ := reader.ReadString('\n')
		publicDomain = strings.TrimSpace(domain)
		if publicDomain == "" {
			log.Println("Skipping GitHub integration - public domain required.")
			return nil
		}
	}

	// Get GitHub token
	if githubToken == "" {
		log.Print("Enter your GitHub personal access token: ")
		token, _ := reader.ReadString('\n')
		githubToken = strings.TrimSpace(token)
		if githubToken == "" {
			log.Println("Skipping GitHub integration - token required for webhook creation.")
			return nil
		}
	}
	return nil
}
