import * as vscode from 'vscode';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
} from 'vscode-languageclient/node';

let client: LanguageClient;
let statusBarItem: vscode.StatusBarItem;
let watchModeActive = false;

export async function activate(context: vscode.ExtensionContext) {
	console.log('Bausteinsicht extension activating...');

	// Create status bar item
	statusBarItem = vscode.window.createStatusBarItem(
		vscode.StatusBarAlignment.Right,
		100
	);
	statusBarItem.text = '$(file-text) Bausteinsicht: Ready';
	statusBarItem.show();
	context.subscriptions.push(statusBarItem);

	// Get LSP server path from config
	const config = vscode.workspace.getConfiguration('bausteinsicht');
	const serverPath = config.get<string>('serverPath') || 'bausteinsicht-lsp';
	const debugMode = config.get<boolean>('debug') || false;

	// Server options: Run LSP server as child process
	const serverOptions: ServerOptions = {
		run: {
			command: serverPath,
			args: debugMode ? ['--debug'] : [],
			transport: TransportKind.stdio,
		},
		debug: {
			command: serverPath,
			args: ['--debug'],
			transport: TransportKind.stdio,
		},
	};

	// Client options
	const clientOptions: LanguageClientOptions = {
		// Activate on any jsonc file (extension is primarily for architecture models)
		documentSelector: [
			{
				scheme: 'file',
				language: 'jsonc',
			},
		],
		synchronize: {
			fileEvents: vscode.workspace.createFileSystemWatcher('**/*architecture*.jsonc'),
		},
		initializationOptions: {
			rootPath: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
		},
	};

	// Create and start LSP client
	client = new LanguageClient(
		'bausteinsicht-lsp',
		'Bausteinsicht Language Server',
		serverOptions,
		clientOptions
	);

	try {
		await client.start();
		statusBarItem.text = '$(check) Bausteinsicht: Connected';
		console.log('Bausteinsicht LSP client started successfully');
	} catch (error) {
		statusBarItem.text = '$(error) Bausteinsicht: Failed to connect';
		const errorMsg = `Failed to start Bausteinsicht LSP server: ${error}`;
		vscode.window.showErrorMessage(errorMsg);
		console.error('Failed to start LSP client:', error);
	}

	// Register commands (always, even if LSP server failed)
	registerCommands(context);
}

function registerCommands(context: vscode.ExtensionContext) {
	// Sync command
	context.subscriptions.push(
		vscode.commands.registerCommand('bausteinsicht.sync', async () => {
			const editor = vscode.window.activeTextEditor;
			if (!editor) {
				vscode.window.showWarningMessage('No editor is active');
				return;
			}
			vscode.window.showInformationMessage('Syncing architecture...');
			// TODO: Call bausteinsicht sync command
		})
	);

	// Validate command
	context.subscriptions.push(
		vscode.commands.registerCommand('bausteinsicht.validate', async () => {
			const editor = vscode.window.activeTextEditor;
			if (!editor) {
				vscode.window.showWarningMessage('No editor is active');
				return;
			}
			vscode.window.showInformationMessage('Validating architecture...');
			// TODO: Call bausteinsicht validate command
		})
	);

	// Health check command
	context.subscriptions.push(
		vscode.commands.registerCommand('bausteinsicht.health', async () => {
			if (!client.isRunning()) {
				vscode.window.showErrorMessage('LSP server is not running');
				return;
			}
			vscode.window.showInformationMessage('✅ Bausteinsicht is healthy');
		})
	);

	// Watch mode toggle
	context.subscriptions.push(
		vscode.commands.registerCommand('bausteinsicht.watchToggle', async () => {
			watchModeActive = !watchModeActive;
			const mode = watchModeActive ? 'enabled' : 'disabled';
			statusBarItem.text = `$(watch) Bausteinsicht: Watch ${mode}`;
			vscode.window.showInformationMessage(`Watch mode ${mode}`);
		})
	);

	// Open in draw.io command
	context.subscriptions.push(
		vscode.commands.registerCommand(
			'bausteinsicht.openInDrawio',
			async (elementId: string, metadata?: any) => {
				const config = vscode.workspace.getConfiguration('bausteinsicht');
				const drawioUrl = config.get<string>('drawioUrl') || 'https://app.diagrams.net';

				// Construct draw.io URL with element ID
				const url = `${drawioUrl}?search=${encodeURIComponent(elementId)}`;
				vscode.env.openExternal(vscode.Uri.parse(url));

				if (metadata) {
					vscode.window.showInformationMessage(
						`Opening ${metadata.kind || 'element'} '${elementId}' in draw.io`
					);
				}
			}
		)
	);
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}
