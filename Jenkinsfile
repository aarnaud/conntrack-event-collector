pipeline {
  agent any
  environment {
       GOROOT = tool(name: 'Go 1.10', type: 'go')
       GOPATH = "${HOME}/golang"
       PATH =  "$PATH:${GOROOT}/bin"
       GONAMESPACE = "${GOPATH}/src/gitlab.com/OpenWifiPortal"
       GOWORKSPACE = "${GOPATH}/src/gitlab.com/OpenWifiPortal/conntrack-event-collector"
  }
  stages {
    stage('Add project in GOPATH') {
      steps {
        sh "mkdir -p $GONAMESPACE"
        sh "ln -sf $WORKSPACE $GOWORKSPACE"
      }
    }
    stage('Build') {
      parallel {
        stage('amd64') {
          steps {
            dir ("$GOWORKSPACE") {
              withEnv(["GOARCH=amd64"]) {
                sh "cd $GOWORKSPACE && ./openwrt/build.sh"
              }
            }
          }
        }
        stage('mipsle') {
          steps {
            dir ("$GOWORKSPACE") {
              withEnv(["GOARCH=mipsle"]) {
                sh "sleep 10"
                sh "cd $GOWORKSPACE && ./openwrt/build.sh"
              }
            }
          }
        }
        stage('arm') {
          steps {
            dir ("$GOWORKSPACE") {
              withEnv(["GOARCH=arm"]) {
                sh "sleep 20"
                sh "cd $GOWORKSPACE && ./openwrt/build.sh"
              }
            }
          }
        }
      }
    }
    stage('Publish artifact') {
      steps {
        archiveArtifacts 'openwrt/build/*.ipk'
      }
    }
  }
}