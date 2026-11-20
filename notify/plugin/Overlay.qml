import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Wayland
import qs.Commons

Item {
  id: root

  property var shell: null
  property var manifest: null
  property bool opened: false
  property bool toastVisible: false
  property string toastTitle: ""
  property string toastBody: ""
  property string toastUrgency: "normal"

  readonly property color ink: Color.menu.text
  readonly property color paper: Color.background
  readonly property color muted: Color.muted
  readonly property color hairline: Qt.rgba(1, 1, 1, 0.18)

  function showToast(title, body, urgency) {
    root.toastTitle = String(title || "").trim();
    root.toastBody = String(body || "").trim();
    root.toastUrgency = String(urgency || "normal").toLowerCase();
    root.toastVisible = root.toastTitle.length > 0 || root.toastBody.length > 0;
    toastTimer.interval = root.toastUrgency === "critical" ? 9000 : 5200;
    toastTimer.restart();
  }

  IpcHandler {
    target: "notify"

    function show(title: string, body: string, urgency: string): string {
      root.showToast(title, body, urgency);
      return "ok";
    }

    function ping(): string {
      return "ok";
    }
  }

  Timer {
    id: toastTimer
    interval: 5200
    repeat: false
    onTriggered: root.toastVisible = false
  }

  PanelWindow {
    id: toastPanel
    visible: root.toastVisible
    implicitWidth: 360
    implicitHeight: root.toastBody.length > 0 ? 58 : 38
    exclusiveZone: 0
    color: "transparent"
    WlrLayershell.layer: WlrLayer.Top
    WlrLayershell.namespace: "omarchy-notify-toast"
    anchors {
      top: true
      right: true
    }
    margins {
      top: 18
      right: 12
    }

    Rectangle {
      anchors.fill: parent
      radius: 3
      color: Qt.rgba(0, 0, 0, 0.76)
      border.width: 1
      border.color: root.toastUrgency === "critical" ? Qt.rgba(1, 1, 1, 0.72) : root.hairline
    }

    Rectangle {
      anchors {
        left: parent.left
        top: parent.top
        bottom: parent.bottom
      }
      width: 2
      color: root.toastUrgency === "critical" ? root.ink : Qt.rgba(1, 1, 1, 0.62)
    }

    Column {
      anchors {
        left: parent.left
        right: parent.right
        verticalCenter: parent.verticalCenter
        leftMargin: 14
        rightMargin: 14
      }
      spacing: 4

      Text {
        width: parent.width
        text: root.toastTitle
        color: root.ink
        elide: Text.ElideRight
        font.family: Style.font.family
        font.pixelSize: 9
        font.weight: Font.Bold
      }

      Text {
        visible: root.toastBody.length > 0
        width: parent.width
        text: root.toastBody
        color: root.muted
        wrapMode: Text.WordWrap
        maximumLineCount: 2
        elide: Text.ElideRight
        font.family: Style.font.family
        font.pixelSize: 8
      }
    }

    MouseArea {
      anchors.fill: parent
      cursorShape: Qt.PointingHandCursor
      onClicked: root.toastVisible = false
    }
  }
}
