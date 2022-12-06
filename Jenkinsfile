pipeline {
  agent any
  stages {
    stage("检出代码") {
      steps {
        checkout([
          $class: 'GitSCM',
          branches: [[name: env.GIT_BUILD_REF]],
          userRemoteConfigs: [[
            url: env.GIT_REPO_URL,
            credentialsId: env.CREDENTIALS_ID
        ]]])
      }
    }
    stage('构建') {
      steps {
        script {
            sh '''
              zip -r src.zip . -x '*.git*' -x '.deploy*' -x 'specs/*'
            '''
        }
      }
    }
    stage('部署') {
      agent {
        docker {
          reuseNode 'true'
          registryUrl 'https://dongfg-docker.pkg.coding.net'
          registryCredentialsId "${env.DOCKER_REGISTRY_CREDENTIALS_ID}"
          image 'serverless-func/docker/fission-cli:1.17.0'
        }
      }
      when {
        branch 'master'
      }
      steps {
        script {
          sh 'fission spec apply --wait'
        }
      }
    }
  }
}