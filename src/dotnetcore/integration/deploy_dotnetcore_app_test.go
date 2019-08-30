package integration_test

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Dotnet Buildpack", func() {
	var app *cutlass.App
	var (
		latest21RuntimeVersion, previous21RuntimeVersion string
		latest21ASPNetVersion, previous21ASPNetVersion   string
		latest21SDKVersion, previous21SDKVersion         string
		latest22SDKVersion, previous22SDKVersion         string
		latest22RuntimeVersion                           string
		latest22ASPNetVersion                            string
	)

	BeforeEach(func() {
		latest21RuntimeVersion = GetLatestDepVersion("dotnet-runtime", "2.1.x", bpDir)
		previous21RuntimeVersion = GetLatestDepVersion("dotnet-runtime", fmt.Sprintf("<%s", latest21RuntimeVersion), bpDir)

		latest21ASPNetVersion = GetLatestDepVersion("dotnet-aspnetcore", "2.1.x", bpDir)
		previous21ASPNetVersion = GetLatestDepVersion("dotnet-aspnetcore", fmt.Sprintf("<%s", latest21ASPNetVersion), bpDir)

		latest21SDKVersion = GetLatestDepVersion("dotnet-sdk", "2.1.x", bpDir)
		previous21SDKVersion = GetLatestDepVersion("dotnet-sdk", fmt.Sprintf("<%s", latest21SDKVersion), bpDir)

		latest22SDKVersion = GetLatestDepVersion("dotnet-sdk", "2.2.x", bpDir)
		previous22SDKVersion = GetLatestDepVersion("dotnet-sdk", fmt.Sprintf("<%s", latest22SDKVersion), bpDir)

		latest22RuntimeVersion = GetLatestDepVersion("dotnet-runtime", "2.2.x", bpDir)

		latest22ASPNetVersion = GetLatestDepVersion("dotnet-aspnetcore", "2.2.x", bpDir)
	})

	AfterEach(func() {
		PrintFailureLogs(app.Name)
		app = DestroyApp(app)
	})

	Context("deploying a source-based app", func() {
		Context("with dotnet-runtime 2.2", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple_2.2_source"))
			})

			It("displays a simple text homepage", func() {
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
			})
		})

		Context("deploying a source-based dotnet 3-preview app", func() {
			Context("with dotnet-runtime 3.0", func() {
				BeforeEach(func() {
					SkipUnlessStack("cflinuxfs3")
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_3_0_app"))
				})

				It("displays a simple text homepage", func() {
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("Welcome"))
				})
			})
		})

		Context("with dotnet sdk 2.1 in global json", func() {
			Context("when the sdk exists", func() {
				BeforeEach(func() {
					app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "source_2.1_global_json_templated"), "global.json", "sdk_version", latest21SDKVersion)
				})

				It("displays a simple text homepage", func() {
					PushAppAndConfirm(app)

					Expect(app.Stdout.String()).To(ContainSubstring(fmt.Sprintf("Installing dotnet-sdk %s", latest21SDKVersion)))
					Expect(app.GetBody("/")).To(ContainSubstring("Hello From Dotnet 2.1"))
				})

			})

			Context("when the sdk is missing", func() {
				BeforeEach(func() {
					app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "source_2.1_global_json_templated"), "global.json", "sdk_version", "2.1.500")
				})

				It("Logs a warning about using default SDK", func() {
					PushAppAndConfirm(app)
					Expect(app.Stdout.String()).To(ContainSubstring("SDK 2.1.500 in global.json is not available"))
					Expect(app.Stdout.String()).To(ContainSubstring("falling back to latest version in version line"))
					Expect(app.GetBody("/")).To(ContainSubstring("Hello From Dotnet 2.1"))
				})
			})
		})

		Context("with buildpack.yml and global.json files", func() {
			Context("when SDK versions don't match", func() {
				BeforeEach(func() {
					app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "with_buildpack_yml_templated"), "global.json", "sdk_version", previous21SDKVersion)
				})

				It("installs the specific version from buildpack.yml instead of global.json", func() {
					app = ReplaceFileTemplate(app.Path, "buildpack.yml", "sdk_version", previous22SDKVersion)
					app.Push()

					Expect(app.Stdout.String()).To(ContainSubstring(fmt.Sprintf("Installing dotnet-sdk %s", previous22SDKVersion)))
				})

				It("installs the floated version from buildpack.yml instead of global.json", func() {
					app = ReplaceFileTemplate(app.Path, "buildpack.yml", "sdk_version", "2.2.x")
					app.Push()

					Expect(app.Stdout.String()).To(ContainSubstring(fmt.Sprintf("Installing dotnet-sdk %s", latest22SDKVersion)))
				})
			})

			Context("when SDK version from buildpack.yml is not available", func() {
				BeforeEach(func() {
					app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "with_buildpack_yml_templated"), "buildpack.yml", "sdk_version", "2.0.0-preview7")
				})

				It("fails due to missing SDK", func() {
					Expect(app.Push()).ToNot(Succeed())

					Eventually(app.Stdout.String).Should(ContainSubstring("SDK 2.0.0-preview7 in buildpack.yml is not available"))
					Eventually(app.Stdout.String).Should(ContainSubstring("Unable to install Dotnet SDK: no match found for 2.0.0-preview7"))
				})
			})
		})

		Context("with node prerendering", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_prerender_node"))
				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("displays a simple text homepage", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("1 + 2 = 3"))
			})
		})

		Context("when RuntimeFrameworkVersion is explicitly defined in csproj", func() {
			BeforeEach(func() {
				app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "source_2.1_explicit_runtime_templated"), "netcoreapp2.csproj", "runtime_version", previous21RuntimeVersion)
				// app = ReplaceFileTemplate(app.Path, "buildpack.yml", "sdk_version", previous21SDKVersion)

				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, using exact runtime", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", previous21RuntimeVersion)))
				Expect(app.GetBody("/")).To(ContainSubstring("Sample pages using ASP.NET Core MVC"))
			})
		})

		Context("when RuntimeFrameworkVersion is floated in csproj", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_2.1_float_runtime"))
				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, using latest patch runtime", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", latest21RuntimeVersion)))
				Expect(app.GetBody("/")).To(ContainSubstring("Sample pages using ASP.NET Core MVC"))
			})
		})

		Context("when the app has Microsoft.AspNetCore.All version 2.1", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_aspnetcore_all_2.1"))
				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, using the TargetFramework for the runtime version and the latest 2.1 patch of dotnet-aspnetcore", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", latest21RuntimeVersion)))
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", latest21ASPNetVersion)))
				Expect(app.GetBody("/")).To(ContainSubstring("Sample pages using ASP.NET Core MVC"))
			})
		})

		Context("when the app has Microsoft.AspNetCore.App version 2.1", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_aspnetcore_app_2.1"))

				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, installing the correct runtime and aspnetcore versions", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", latest21ASPNetVersion)))
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", latest21RuntimeVersion)))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))

				By("accepts SIGTERM and exits gracefully")
				Expect(app.Stop()).To(Succeed())
				Eventually(func() string { return app.Stdout.String() }, 30*time.Second, 1*time.Second).Should(ContainSubstring("Goodbye, cruel world!"))
			})
		})

		Context("when the app has Microsoft.AspNetCore.All version 2.0", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_2.0"))

				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, installing the a roll forward runtime and aspnetcore versions", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", latest22RuntimeVersion)))
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", latest22ASPNetVersion)))
				Expect(app.GetBody("/")).To(ContainSubstring("Sample pages using ASP.NET Core MVC"))
			})
		})

		Context("with AssemblyName specified", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_dot_in_name"))
				app.Memory = "1G"
				app.Disk = "2G"
			})

			It("successfully pushes an app with an AssemblyName", func() {
				PushAppAndConfirm(app)
			})
		})

		Context("with libgdiplus", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "uses_libgdiplus"))
			})

			It("displays a simple text homepage", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Installing libgdiplus"))
			})
		})

		Context("without libgdiplus", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "source_aspnetcore_app_2.1"))
			})

			It("displays a simple text homepage", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).NotTo(ContainSubstring("Installing libgdiplus"))
			})
		})
	})

	Context("deploying an FDD app", func() {
		Context("with Microsoft.AspNetCore.App 2.1", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "fdd_aspnetcore_2.1"))

				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, and floats the runtime and aspnetcore versions by default", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", latest21ASPNetVersion)))
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", latest21RuntimeVersion)))

				By("accepts SIGTERM and exits gracefully")
				Expect(app.Stop()).ToNot(HaveOccurred())
				Eventually(func() string { return app.Stdout.String() }, 30*time.Second, 1*time.Second).Should(ContainSubstring("Goodbye, cruel world!"))
			})
		})

		Context("with Microsoft.AspNetCore.App 3.0-preview", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "fdd_3.0_preview"))

				app.Disk = "2G"
				app.Memory = "2G"
			})

			It("publishes and runs, the preview versions of the runtime and aspnetcore", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", "3.0")))
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-runtime %s", "3.0")))
			})
		})

		Context("with Microsoft.AspNetCore.App 2.1 and applyPatches false", func() {
			BeforeEach(func() {
				app = ReplaceFileTemplate(filepath.Join(bpDir, "fixtures", "fdd_apply_patches_false_2.1_templated"), "dotnet.runtimeconfig.json", "framework_version", previous21ASPNetVersion)
			})

			It("installs the exact version of dotnet-aspnetcore from the runtimeconfig.json", func() {
				PushAppAndConfirm(app)
				Eventually(app.Stdout.String()).Should(ContainSubstring(fmt.Sprintf("Installing dotnet-aspnetcore %s", previous21ASPNetVersion)))
			})
		})

		Context("with libgdiplus", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "uses_libgdiplus", "bin", "Debug", "netcoreapp2.2", "publish"))
			})

			It("displays a simple text homepage", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Installing libgdiplus"))
			})
		})
	})

	Context("deploying a self contained msbuild app with RuntimeIdentfier", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "self_contained_msbuild"))
		})

		It("displays a simple text homepage", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(MatchRegexp("Removing dotnet-sdk"))

			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
	})

	Context("deploying an app with comments in the runtimeconfig.json", func() {
		It("should deploy", func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "runtimeconfig_with_comments"))
			PushAppAndConfirm(app)
		})
	})
})
