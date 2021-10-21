module github.com/tjohn327/deadline_aware_multipath

go 1.16

replace github.com/netsec-ethz/scion-apps v0.4.0 => github.com/netsys-lab/scion-apps v0.1.1-0.20210929142559-a9ca2024f287

require github.com/netsec-ethz/scion-apps v0.4.0

require (
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/klauspost/reedsolomon v1.9.13
)
