/*
Copyright © 2024 Anton Brekhov <anton@abrekhov.ru>

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
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hstream",
	Short: "WebRTC video and audio streaming platform",
	Long: `HyperStream - broadcast video and audio streams via WebRTC.
Viewers can watch in any modern browser without installing anything.`,
}

// Execute runs the root command.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.AddCommand(broadcastCmd)
}
