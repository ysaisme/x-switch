#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

@interface AppDelegate : NSObject <NSApplicationDelegate, WKNavigationDelegate, NSMenuDelegate>
@property (strong) NSWindow *window;
@property (strong) WKWebView *webView;
@property (strong) NSStatusItem *statusItem;
@property (copy) NSString *mswitchPath;
@property (copy) NSString *activeProfile;
@property (strong) NSArray *cachedProfiles;
@property (strong) NSArray *cachedSites;
@end

@implementation AppDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    self.mswitchPath = [NSBundle.mainBundle.resourcePath stringByAppendingPathComponent:@"mswitch"];
    self.cachedProfiles = @[];
    self.cachedSites = @[];
    self.activeProfile = @"";

    if (![[NSFileManager defaultManager] fileExistsAtPath:self.mswitchPath]) {
        NSAlert *alert = [[NSAlert alloc] init];
        alert.messageText = @"mswitch binary not found";
        alert.informativeText = [NSString stringWithFormat:@"Expected: %@", self.mswitchPath];
        [alert runModal];
        [NSApp terminate:nil];
        return;
    }

    [self ensureServerRunning];
    [self setupStatusBar];
    [self setupWindow];
    [self waitForServerAndLoad];
    [self refreshMenuData];
}

#pragma mark - Status Bar

- (void)setupStatusBar {
    self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    self.statusItem.button.image = [self statusBarIcon];
    self.statusItem.button.image.template = YES;

    NSMenu *menu = [[NSMenu alloc] init];
    menu.delegate = self;
    self.statusItem.menu = menu;
    [self buildMenu:menu];
}

- (NSImage *)statusBarIcon {
    NSString *iconPath = [NSBundle.mainBundle.resourcePath stringByAppendingPathComponent:@"icon-menu.png"];
    NSImage *image = [[NSImage alloc] initWithContentsOfFile:iconPath];
    if (!image) {
        image = [[NSImage alloc] initWithSize:NSMakeSize(20, 20)];
        [image lockFocus];
        NSBezierPath *circle = [NSBezierPath bezierPathWithOvalInRect:NSMakeRect(2, 2, 16, 16)];
        [[NSColor blackColor] setFill];
        [circle fill];
        NSDictionary *attrs = @{
            NSFontAttributeName: [NSFont boldSystemFontOfSize:11],
            NSForegroundColorAttributeName: [NSColor whiteColor]
        };
        NSAttributedString *text = [[NSAttributedString alloc] initWithString:@"M" attributes:attrs];
        NSSize textSize = text.size;
        NSPoint textPoint = NSMakeRect(0, 0, 20, 20).origin;
        textPoint = NSMakePoint((20 - textSize.width) / 2, (20 - textSize.height) / 2);
        [text drawAtPoint:textPoint];
        [image unlockFocus];
    }
    image.template = YES;
    return image;
}

- (void)buildMenu:(NSMenu *)menu {
    [menu removeAllItems];

    NSMenuItem *titleItem = [[NSMenuItem alloc] initWithTitle:@"mswitch" action:nil keyEquivalent:@""];
    titleItem.enabled = NO;
    [menu addItem:titleItem];
    [menu addItem:[NSMenuItem separatorItem]];

    if (self.cachedProfiles.count > 0) {
        NSMenuItem *profileHeader = [[NSMenuItem alloc] initWithTitle:@"Profile" action:nil keyEquivalent:@""];
        profileHeader.enabled = NO;
        [menu addItem:profileHeader];

        for (NSDictionary *p in self.cachedProfiles) {
            NSString *name = p[@"name"];
            BOOL isActive = [name isEqualToString:self.activeProfile];
            NSString *title = isActive ? [NSString stringWithFormat:@"● %@", name] : [NSString stringWithFormat:@"    %@", name];
            NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:@selector(switchProfile:) keyEquivalent:@""];
            item.representedObject = name;
            item.target = self;
            if (isActive) {
                item.state = NSControlStateValueOn;
            }
            [menu addItem:item];
        }
        [menu addItem:[NSMenuItem separatorItem]];
    }

    if (self.cachedSites.count > 0) {
        NSMenuItem *siteHeader = [[NSMenuItem alloc] initWithTitle:@"快速切换站点" action:nil keyEquivalent:@""];
        siteHeader.enabled = NO;
        [menu addItem:siteHeader];

        for (NSDictionary *s in self.cachedSites) {
            NSString *siteId = s[@"id"];
            NSString *siteName = s[@"name"];
            NSString *title = [NSString stringWithFormat:@"    → %@", siteName];
            NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:@selector(switchSite:) keyEquivalent:@""];
            item.representedObject = siteId;
            item.target = self;
            [menu addItem:item];
        }
        [menu addItem:[NSMenuItem separatorItem]];
    }

    NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"打开主窗口" action:@selector(openMainWindow:) keyEquivalent:@""];
    openItem.target = self;
    [menu addItem:openItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"退出 mswitch" action:@selector(terminateApp:) keyEquivalent:@"q"];
    quitItem.target = self;
    [menu addItem:quitItem];
}

- (void)menuWillOpen:(NSMenu *)menu {
    [self refreshMenuData];
}

- (void)refreshMenuData {
    dispatch_group_t group = dispatch_group_create();

    dispatch_group_enter(group);
    [self fetchJSON:@"/api/v1/routing/current" completion:^(NSDictionary *data) {
        if (data) {
            self.activeProfile = data[@"active_profile"] ?: @"";
        }
        dispatch_group_leave(group);
    }];

    dispatch_group_enter(group);
    [self fetchJSON:@"/api/v1/profiles" completion:^(NSDictionary *data) {
        if (data) {
            self.cachedProfiles = data[@"profiles"] ?: @[];
        }
        dispatch_group_leave(group);
    }];

    dispatch_group_enter(group);
    [self fetchJSON:@"/api/v1/sites" completion:^(NSDictionary *data) {
        if (data) {
            self.cachedSites = data[@"sites"] ?: @[];
        }
        dispatch_group_leave(group);
    }];

    dispatch_group_notify(group, dispatch_get_main_queue(), ^{
        [self buildMenu:self.statusItem.menu];
    });
}

- (void)fetchJSON:(NSString *)path completion:(void (^)(NSDictionary *))completion {
    NSURL *url = [NSURL URLWithString:[NSString stringWithFormat:@"http://127.0.0.1:9091%@", path]];
    NSURLSessionDataTask *task = [NSURLSession.sharedSession
        dataTaskWithURL:url
        completionHandler:^(NSData *data, NSURLResponse *resp, NSError *err) {
            if (!data) {
                dispatch_async(dispatch_get_main_queue(), ^{ completion(nil); });
                return;
            }
            NSDictionary *json = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
            dispatch_async(dispatch_get_main_queue(), ^{ completion(json); });
        }];
    [task resume];
}

- (void)postSwitch:(NSDictionary *)body {
    NSURL *url = [NSURL URLWithString:@"http://127.0.0.1:9091/api/v1/routing/switch"];
    NSMutableURLRequest *req = [NSMutableURLRequest requestWithURL:url];
    req.HTTPMethod = @"POST";
    [req setValue:@"application/json" forHTTPHeaderField:@"Content-Type"];
    req.HTTPBody = [NSJSONSerialization dataWithJSONObject:body options:0 error:nil];

    [[NSURLSession.sharedSession dataTaskWithRequest:req completionHandler:^(NSData *data, NSURLResponse *resp, NSError *err) {
        if ([(NSHTTPURLResponse *)resp statusCode] == 200) {
            dispatch_async(dispatch_get_main_queue(), ^{
                [self refreshMenuData];
            });
        }
    }] resume];
}

- (void)switchProfile:(NSMenuItem *)sender {
    NSString *name = sender.representedObject;
    if (!name) return;
    [self postSwitch:@{@"profile": name}];
}

- (void)switchSite:(NSMenuItem *)sender {
    NSString *siteId = sender.representedObject;
    if (!siteId) return;
    [self postSwitch:@{@"site": siteId}];
}

- (void)openMainWindow:(NSMenuItem *)sender {
    if (!self.window) {
        [self setupWindow];
        [self waitForServerAndLoad];
    }
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

- (void)terminateApp:(NSMenuItem *)sender {
    [NSApp terminate:nil];
}

#pragma mark - Window

- (void)setupWindow {
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
}

#pragma mark - Server

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

#pragma mark - App Lifecycle

- (void)applicationWillTerminate:(NSNotification *)notification {
    NSTask *process = [[NSTask alloc] init];
    process.executableURL = [NSURL fileURLWithPath:self.mswitchPath];
    process.arguments = @[@"stop"];
    [process launch];
    [process waitUntilExit];
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)flag {
    if (!flag || !self.window) {
        if (!self.window) {
            [self setupWindow];
            [self waitForServerAndLoad];
        }
        [self.window makeKeyAndOrderFront:nil];
    }
    return YES;
}

#pragma mark - WKNavigationDelegate

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
