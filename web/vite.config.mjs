import { defineConfig } from "vite";
export default defineConfig({
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "language-data",
              test: /internal[\\/]langdata[\\/]registry\.json(?:\?|$)/,
            },
          ],
        },
      },
    },
  },
});
