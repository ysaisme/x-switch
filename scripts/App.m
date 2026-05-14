#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

@interface AppDelegate : NSObject <NSApplicationDelegate, WKNavigationDelegate>
@property (strong) NSWindow *window;
@property (strong) WKWebView *webView;
@property (copy) NSString *mswitchPath;
@end

@implementation AppDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    self.mswitchPath = [NSBundle.mainBundle.resourcePath stringByAppendingPathComponent:@"mswitch"];

    if (![[NSFileManager defaultManager] fileExistsAtPath:self.mswitchPath]) {
        NSAlert *alert = [[NSAlert alloc] init];
        alert.messageText = @"mswitch binary not found";
        alert.informativeText = [NSString stringWithFormat:@"Expected: %@", self.mswitchPath];
        [alert runModal];
        [NSApp terminate:nil];
        return;
    }

    [self ensureServerRunning];

    NSRect frame = NSMakeRect(0, 0, 1200, 800);
    self.window = [[NSWindow alloc] initWithContentRect:frame
                                             styleMask:(NSWindowStyleMaskTitled |
                                                        NSWindowStyleMaskClosable |
                                                        NSWindowStyleMaskMiniaturizable |
                                                        NSWindowStyleMaskResizable)
                                               backing:NSBackingStoreBuffered
                                                  defer:NO];
    [self.window center];
    self.window.title = @"mswitch";
    self.window.minSize = NSMakeSize(800, 600);
    self.window.titleVisibility = NSWindowTitleHidden;
    self.window.titlebarAppearsTransparent = YES;
    self.window.movableByWindowBackground = YES;

    WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
    self.webView = [[WKWebView alloc] initWithFrame:CGRectZero configuration:config];
    self.webView.navigationDelegate = self;
    self.window.contentView = self.webView;

    [self.window makeKeyAndOrderFront:nil];
    [self waitForServerAndLoad];
}

- (void)ensureServerRunning {
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    __block BOOL running = NO;

    NSURLSessionDataTask *task = [NSURLSession.sharedSession
        dataTaskWithURL:[NSURL URLWithString:@"http://127.0.0.1:9091/api/v1/health"]
        completionHandler:^(NSData *data, NSURLResponse *resp, NSError *err) {
            if ([(NSHTTPURLResponse *)resp statusCode] == 200) running = YES;
            dispatch_semaphore_signal(sem);
        }];
    [task resume];
    dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 3 * NSEC_PER_SEC));

    if (running) return;

    NSTask *process = [[NSTask alloc] init];
    process.executableURL = [NSURL fileURLWithPath:self.mswitchPath];
    process.arguments = @[@"start"];
    [process launch];
    [process waitUntilExit];
}

- (void)waitForServerAndLoad {
    __block int attempts = 0;
    __weak typeof(self) weakSelf = self;

    void (^check)(void) = ^{
        attempts++;
        NSURLSessionDataTask *task = [NSURLSession.sharedSession
            dataTaskWithURL:[NSURL URLWithString:@"http://127.0.0.1:9091/api/v1/health"]
            completionHandler:^(NSData *data, NSURLResponse *resp, NSError *err) {
                if ([(NSHTTPURLResponse *)resp statusCode] == 200) {
                    dispatch_async(dispatch_get_main_queue(), ^{
                        typeof(self) strongSelf = weakSelf;
                        if (!strongSelf) return;
                        NSURL *webURL = [NSURL URLWithString:@"http://127.0.0.1:9091"];
                        [strongSelf.webView loadRequest:[NSURLRequest requestWithURL:webURL]];
                    });
                } else if (attempts < 60) {
                    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 500 * NSEC_PER_MSEC), dispatch_get_main_queue(), ^{
                        typeof(self) strongSelf = weakSelf;
                        if (!strongSelf) return;
                        check();
                    });
                }
            }];
        [task resume];
    };
    check();
}

- (void)applicationWillTerminate:(NSNotification *)notification {
    NSTask *process = [[NSTask alloc] init];
    process.executableURL = [NSURL fileURLWithPath:self.mswitchPath];
    process.arguments = @[@"stop"];
    [process launch];
    [process waitUntilExit];
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)flag {
    if (!flag) [self.window makeKeyAndOrderFront:nil];
    return YES;
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationAction:(WKNavigationAction *)navigationAction
    decisionHandler:(void (^)(WKNavigationActionPolicy))decisionHandler {
    NSString *scheme = navigationAction.request.URL.scheme;
    if ([scheme isEqualToString:@"http"] || [scheme isEqualToString:@"https"] || [scheme isEqualToString:@"about"]) {
        decisionHandler(WKNavigationActionPolicyAllow);
    } else {
        decisionHandler(WKNavigationActionPolicyCancel);
    }
}

@end

int main(int argc, const char *argv[]) {
    NSApplication *app = [NSApplication sharedApplication];
    [app setActivationPolicy:NSApplicationActivationPolicyRegular];
    AppDelegate *delegate = [[AppDelegate alloc] init];
    [app setDelegate:delegate];
    [app activateIgnoringOtherApps:YES];
    [app run];
    return 0;
}
