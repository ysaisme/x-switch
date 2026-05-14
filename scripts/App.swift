import Cocoa
import WebKit

@NSApplicationMain
class AppDelegate: NSObject, NSApplicationDelegate {
    var window: NSWindow!
    var webView: WKWebView!
    var mswitchPath: String!
    let webURL = "http://127.0.0.1:9091"
    let healthURL = "http://127.0.0.1:9091/api/v1/health"

    func applicationDidFinishLaunching(_ notification: Notification) {
        mswitchPath = Bundle.main.resourcePath! + "/mswitch"

        if !FileManager.default.fileExists(atPath: mswitchPath) {
            let alert = NSAlert()
            alert.messageText = "mswitch binary not found"
            alert.informativeText = "Expected: \(mswitchPath!)"
            alert.alertStyle = .critical
            alert.runModal()
            NSApp.terminate(nil)
            return
        }

        ensureServerRunning()

        let rect = NSRect(x: 0, y: 0, width: 1200, height: 800)
        window = NSWindow(
            contentRect: rect,
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.center()
        window.title = "mswitch"
        window.minSize = NSSize(width: 800, height: 600)
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.isMovableByWindowBackground = true

        let config = WKWebViewConfiguration()
        webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = self
        window.contentView = webView

        window.makeKeyAndOrderFront(nil)

        waitForServerAndLoad()
    }

    func ensureServerRunning() {
        let checkUrl = URL(string: healthURL)!
        let sem = DispatchSemaphore(value: 0)
        var running = false

        URLSession.shared.dataTask(with: checkUrl) { _, resp, _ in
            if let http = resp as? HTTPURLResponse, http.statusCode == 200 {
                running = true
            }
            sem.signal()
        }.resume()

        sem.wait()

        if running { return }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: mswitchPath)
        process.arguments = ["start"]
        try? process.run()
        process.waitUntilExit()
    }

    func waitForServerAndLoad() {
        var attempts = 0
        func check() {
            attempts += 1
            let url = URL(string: healthURL)!
            URLSession.shared.dataTask(with: url) { _, resp, _ in
                if let http = resp as? HTTPURLResponse, http.statusCode == 200 {
                    DispatchQueue.main.async {
                        self.webView.load(URLRequest(url: URL(string: self.webURL)!))
                    }
                } else if attempts < 60 {
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { check() }
                }
            }.resume()
        }
        check()
    }

    func applicationWillTerminate(_ notification: Notification) {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: mswitchPath)
        process.arguments = ["stop"]
        try? process.run()
        process.waitUntilExit()
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        if !flag {
            window.makeKeyAndOrderFront(nil)
        }
        return true
    }
}

extension AppDelegate: WKNavigationDelegate {
    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        if let url = navigationAction.request.url {
            let scheme = url.scheme ?? ""
            if scheme == "http" || scheme == "https" || scheme == "about" {
                decisionHandler(.allow)
                return
            }
        }
        decisionHandler(.cancel)
    }
}
