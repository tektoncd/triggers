# Tekton Repo CI/CD

_Why does Tekton triggers have a folder called `tekton`? Cuz we think it would
be cool if the `tekton` folder were the place to look for CI/CD logic in most
repos!_

We use Tekton Pipelines to build, test and release Tekton Triggers!

This directory contains the
[`Tasks`](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md) and
[`Pipelines`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md)
that we use.

The Pipelines and Tasks in this folder are used for:

1. [Manually creating official releases from the official cluster](#create-an-official-release)

To start from scratch and use these Pipelines and Tasks:

1. [Install Tekton](https://github.com/tektoncd/pipeline/blob/master/tekton/README.md#install-tekton)
1. [Setup the Tasks and Pipelines](https://github.com/tektoncd/pipeline/blob/master/tekton/README.md#install-tasks-and-pipelines)
1. [Create the required service account + secrets](https://github.com/tektoncd/pipeline/blob/master/tekton/README.md#service-account-and-secrets)

## Create an official release

To create an official release, follows the steps in the [release-cheat-sheet](./release-cheat-sheet.md)
