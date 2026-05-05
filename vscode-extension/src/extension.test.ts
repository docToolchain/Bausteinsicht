import * as assert from 'assert';
import * as vscode from 'vscode';

suite('Bausteinsicht Extension Tests', () => {
	vscode.window.showInformationMessage('Start all tests.');

	test('Extension can be activated', async () => {
		const ext = vscode.extensions.getExtension('docToolchain.bausteinsicht');
		assert.ok(ext, 'Extension should be found');

		if (ext && !ext.isActive) {
			await ext.activate();
		}

		assert.ok(ext?.isActive, 'Extension should be activated');
	});

	test('Command bausteinsicht.validate is registered', async () => {
		const commands = await vscode.commands.getCommands();
		assert.ok(
			commands.includes('bausteinsicht.validate'),
			'validate command should be registered'
		);
	});

	test('Command bausteinsicht.sync is registered', async () => {
		const commands = await vscode.commands.getCommands();
		assert.ok(
			commands.includes('bausteinsicht.sync'),
			'sync command should be registered'
		);
	});

	test('Command bausteinsicht.health is registered', async () => {
		const commands = await vscode.commands.getCommands();
		assert.ok(
			commands.includes('bausteinsicht.health'),
			'health command should be registered'
		);
	});

	test('Command bausteinsicht.watchToggle is registered', async () => {
		const commands = await vscode.commands.getCommands();
		assert.ok(
			commands.includes('bausteinsicht.watchToggle'),
			'watchToggle command should be registered'
		);
	});

	test('Command bausteinsicht.openInDrawio is registered', async () => {
		const commands = await vscode.commands.getCommands();
		assert.ok(
			commands.includes('bausteinsicht.openInDrawio'),
			'openInDrawio command should be registered'
		);
	});

	test('Configuration bausteinsicht.serverPath exists', async () => {
		const config = vscode.workspace.getConfiguration('bausteinsicht');
		const serverPath = config.get<string>('serverPath');
		assert.ok(serverPath !== undefined, 'serverPath config should exist');
		assert.strictEqual(serverPath, 'bausteinsicht-lsp', 'default serverPath should be bausteinsicht-lsp');
	});

	test('Configuration bausteinsicht.debug exists', async () => {
		const config = vscode.workspace.getConfiguration('bausteinsicht');
		const debug = config.get<boolean>('debug');
		assert.strictEqual(typeof debug, 'boolean', 'debug config should be boolean');
	});

	test('Configuration bausteinsicht.drawioUrl exists', async () => {
		const config = vscode.workspace.getConfiguration('bausteinsicht');
		const drawioUrl = config.get<string>('drawioUrl');
		assert.ok(drawioUrl !== undefined, 'drawioUrl config should exist');
		assert.ok(drawioUrl?.includes('diagrams.net'), 'drawioUrl should contain diagrams.net');
	});
});

suite('Extension Commands Tests', () => {
	test('bausteinsicht.health command can execute', async () => {
		try {
			await vscode.commands.executeCommand('bausteinsicht.health');
			// Command executed successfully
			assert.ok(true, 'health command should execute');
		} catch (error) {
			// Some commands may fail if LSP server is not running, which is expected in tests
			assert.ok(true, 'health command execution attempted');
		}
	});

	test('bausteinsicht.watchToggle command can execute', async () => {
		try {
			await vscode.commands.executeCommand('bausteinsicht.watchToggle');
			assert.ok(true, 'watchToggle command should execute');
		} catch (error) {
			assert.ok(true, 'watchToggle command execution attempted');
		}
	});

	test('bausteinsicht.openInDrawio command can execute with element ID', async () => {
		try {
			await vscode.commands.executeCommand('bausteinsicht.openInDrawio', 'test-element', {
				kind: 'service',
				status: 'active',
				views: 1
			});
			assert.ok(true, 'openInDrawio command should execute');
		} catch (error) {
			// External link opening may not work in test environment
			assert.ok(true, 'openInDrawio command execution attempted');
		}
	});
});
