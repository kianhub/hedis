import Metro from 'metro';
	const config = await Metro.loadConfig();
	config.resetCache = true;
	await Metro.runBuild(config, {
		entry: 'package_entry.js',
		platform: 'ios',
		minify: false,
		out: 'package_out_0.71.js'
	});