/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ['./index.html', './src/**/*.{svelte,ts,js}'],
    darkMode: 'class',
    theme: {
        extend: {
            fontFamily: {
                display: ['"Space Grotesk"', 'ui-sans-serif', 'system-ui'],
                body: ['"IBM Plex Sans"', 'ui-sans-serif', 'system-ui'],
            },
            colors: {
                slate: {
                    950: '#0b1016',
                },
            },
            boxShadow: {
                glass: '0 10px 30px rgba(0,0,0,0.25)',
            },
        },
    },
    plugins: [],
}
