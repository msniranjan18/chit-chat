### File Summary

-----------------------------------------------------------------------------------
File	        |   Purpose		        | DevOnly | Prod  |   When Created
-----------------------------------------------------------------------------------
.air.toml		| Hot reload config	    |   ✅	 | ❌	|   Early development
Dockerfile.dev	| Dev container	        |   ✅	 | ❌	|   Team/docker setup
.golangci.yml	| Code quality	        |   ✅	 | ✅(CI)|   Project start
Dockerfile	    | Production container	|   ❌	 | ✅	|   Deployment ready
Makefile	    | Task automation	    |   ✅	 | ✅	|   Project start
.env	        | Environment vars	    |   ✅	 | ✅	|   Project start