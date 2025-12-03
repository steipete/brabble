#!/usr/bin/env node
import { execa } from 'execa';
import chalk from 'chalk';
import { existsSync } from 'fs';
import { join } from 'path';

const BIN = join(process.cwd(), 'bin', 'brabble');

function printHelp() {
  console.log(chalk.bold.blue('Brabble CLI helper (pnpm brabble)'));
  console.log(chalk.dim('Builds the Go binary if needed, then runs it.'));
  console.log('');
  console.log(chalk.bold('Usage:'));
  console.log('  pnpm brabble             ', chalk.dim('build + serve (foreground)'));
  console.log('  pnpm brabble <args...>   ', chalk.dim('build + run ./bin/brabble <args>'));
  console.log('  pnpm brabble --help      ', chalk.dim('this help'));
  console.log('  pnpm brabble --version   ', chalk.dim('show brabble version'));
  console.log('');
  console.log(chalk.bold('Common args:'));
  console.log('  start|stop|restart|status|tail-log|list-mics|set-mic|doctor|setup|models ...');
}

async function ensureBuilt() {
  if (existsSync(BIN)) return;
  console.log(chalk.yellow('Building brabble binary...'));
  await execa('go', ['build', '-o', BIN, './cmd/brabble'], { stdio: 'inherit' });
}

async function main() {
  const args = process.argv.slice(2);
  if (args.includes('--help') || args.includes('-h')) {
    printHelp();
    return;
  }
  if (args.includes('--version') || args.includes('-v')) {
    await ensureBuilt();
    const { stdout } = await execa(BIN, ['--version']);
    console.log(stdout.trim());
    return;
  }
  await ensureBuilt();
  const runArgs = args.length === 0 ? ['serve'] : args;
  const child = execa(BIN, runArgs, { stdio: 'inherit' });
  child.on('exit', code => process.exit(code ?? 0));
}

main().catch(err => {
  console.error(chalk.red('Error:'), err.message);
  process.exit(1);
});
