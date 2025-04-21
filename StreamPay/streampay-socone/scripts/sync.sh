#!/bin/bash
#-------------------------------------------
# Source repository information 
# Origin github
#-------------------------------------------
GITHUB_ORG=OmniFlix
GITHUB_REPO=streampay
GITHUB_BRANCH=main

#-------------------------------------------
# Destination repository information 
# Our gitlab
#-------------------------------------------
GITLAB_ORG=xplor/x-alpha/xblockchain
GITLAB_REPO=streampay
GITLAB_BRANCH=socone

git clone ssh://git@gitlab.soc.one:2222/$GITLAB_ORG/${GITLAB_REPO}.git
cd $GITLAB_REPO
git checkout $GITLAB_BRANCH
git remote -v
git remote add upstream https://github.com/$GITHUB_ORG/${GITHUB_REPO}.git
git remote -v
git fetch upstream
git merge upstream/$GITHUB_BRANCH
git push