package analyzer

// This file is used to gather yaml config message for all
// analyzers.
//
// All the config type can be RawMessage in AnaConfig

/* aggregation:
name: string
help: string
level: int(0-3)
constLabels: map[string]string
duration: string(must fit in time.ParseDuration)
type: string(max/min)
*/

/* quatile:
name: string
help: string
level: int(0-3)
constLabels: map[string]string
targets: list[float](0-1)
*/
