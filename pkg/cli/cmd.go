package cli

import (
	"io"

	"github.com/spf13/cobra"

	koliutil "github.com/kolibox/koli/pkg/cli/util"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	kubecmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	cmdconfig "k8s.io/kubernetes/pkg/kubectl/cmd/config"
	"k8s.io/kubernetes/pkg/kubectl/cmd/rollout"
	"k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	"k8s.io/kubernetes/pkg/util/flag"
)

const (
	// PossibleResourceTypes help description
	PossibleResourceTypes = `Possible resource types include (case insensitive): 
pods (aka 'po'), deployments (aka 'deploy'), events (aka 'ev'), stacks, 
horizontalpodautoscalers (aka 'hpa'), configmaps (aka 'cm') and secrets.`

	bashCompletionFunc = `# call kubectl get $1,
__kubectl_override_flag_list=(kubeconfig cluster user context namespace server)
__kubectl_override_flags()
{
    local ${__kubectl_override_flag_list[*]} two_word_of of
    for w in "${words[@]}"; do
        if [ -n "${two_word_of}" ]; then
            eval "${two_word_of}=\"--${two_word_of}=\${w}\""
            two_word_of=
            continue
        fi
        for of in "${__kubectl_override_flag_list[@]}"; do
            case "${w}" in
                --${of}=*)
                    eval "${of}=\"--${of}=\${w}\""
                    ;;
                --${of})
                    two_word_of="${of}"
                    ;;
            esac
        done
        if [ "${w}" == "--all-namespaces" ]; then
            namespace="--all-namespaces"
        fi
    done
    for of in "${__kubectl_override_flag_list[@]}"; do
        if eval "test -n \"\$${of}\""; then
            eval "echo \${${of}}"
        fi
    done
}

__kubectl_get_namespaces()
{
    local template kubectl_out
    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
    if kubectl_out=$(kubectl get -o template --template="${template}" namespace 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out[*]}" -- "$cur" ) )
    fi
}

__kubectl_parse_get()
{
    local template
    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
    local kubectl_out
    if kubectl_out=$(kubectl get $(__kubectl_override_flags) -o template --template="${template}" "$1" 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out[*]}" -- "$cur" ) )
    fi
}

__kubectl_get_resource()
{
    if [[ ${#nouns[@]} -eq 0 ]]; then
        return 1
    fi
    __kubectl_parse_get "${nouns[${#nouns[@]} -1]}"
}

__kubectl_get_resource_pod()
{
    __kubectl_parse_get "pod"
}

__kubectl_get_resource_rc()
{
    __kubectl_parse_get "rc"
}

# $1 is the name of the pod we want to get the list of containers inside
__kubectl_get_containers()
{
    local template
    template="{{ range .spec.containers  }}{{ .name }} {{ end }}"
    __debug "${FUNCNAME} nouns are ${nouns[*]}"

    local len="${#nouns[@]}"
    if [[ ${len} -ne 1 ]]; then
        return
    fi
    local last=${nouns[${len} -1]}
    local kubectl_out
    if kubectl_out=$(kubectl get $(__kubectl_namespace_flag) -o template --template="${template}" pods "${last}" 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out[*]}" -- "$cur" ) )
    fi
}

# Require both a pod and a container to be specified
__kubectl_require_pod_and_container()
{
    if [[ ${#nouns[@]} -eq 0 ]]; then
        __kubectl_parse_get pods
        return 0
    fi;
    __kubectl_get_containers
    return 0
}

__custom_func() {
    case ${last_command} in
        kubectl_get | kubectl_describe | kubectl_delete | kubectl_label | kubectl_stop | kubectl_edit | kubectl_patch |\
        kubectl_annotate | kubectl_expose)
            __kubectl_get_resource
            return
            ;;
        kubectl_logs)
            __kubectl_require_pod_and_container
            return
            ;;
        kubectl_exec)
            __kubectl_get_resource_pod
            return
            ;;
        kubectl_rolling-update)
            __kubectl_get_resource_rc
            return
            ;;
        *)
            ;;
    esac
}
`

	// If you add a resource to this list, please also take a look at pkg/kubectl/kubectl.go
	// and add a short forms entry in expandResourceShortcut() when appropriate.
	validResources = `Valid resource types include:
   * configmaps (aka 'cm')
   * deployments (aka 'deploy')
   * events (aka 'ev')
   * horizontalpodautoscalers (aka 'hpa')
   * namespaces (aka 'ns')
   * pods (aka 'po')
   * services (aka 'svc')
`
	usageTemplate = `{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Subcommands, use 'koli [subcommand] -h/--help' to learn more:
{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}

Usage:{{if .Runnable}}
  {{if .HasFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{ if .HasSubCommands }}
  {{ .CommandPath}} [command]

Use 'git push {{.CommandPath}} master' to deploy to an application.{{end}}
`
	helpTemplate = `{{with or .Long .Short }}{{. | trim}}{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
)

// NewKubectlCommand creates the `kubectl` command and its nested children.
func NewKubectlCommand(f *koliutil.Factory, in io.Reader, out, err io.Writer) *cobra.Command {
	// Parent command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   "koli",
		Short: "Koli command-line controls your cluster apps",
		// Long: "Koli command-line controls your cluster apps",
		Run: runHelp,
		BashCompletionFunction: bashCompletionFunc,
	}
	// cmds.SetHelpTemplate(helpTemplate)
	// cmds.SetUsageTemplate(usageTemplate)

	comm := &koliutil.CommandParams{Factory: f, In: in, Out: out, Err: err}

	// ./koli options (flags)
	f.BindFlags(cmds.PersistentFlags())
	f.BindExternalFlags(cmds.PersistentFlags())

	// cmds.PersistentFlags().AddFlagSet(globalFlags)
	// cmds.SetHelpCommand(kubecmd.NewCmdHelp(f, out))

	// From this point and forward we get warnings on flags that contain "_" separators
	cmds.SetGlobalNormalizationFunc(flag.WarnWordSepNormalizeFunc)

	groups := templates.CommandGroups{
		{
			Message: "Primary commands, use 'koli [command] -h/--help' to learn more:\n",
			Commands: []*cobra.Command{
				NewCmdCreate(f, out),
				NewCmdDelete(f, out),
				NewCmdDescribe(f, out, err),
				NewCmdLabel(f, out),
				NewCmdGet(comm),
			},
		},
		{
			Message: "Subcommands:\n",
			Commands: []*cobra.Command{
				kubecmd.NewCmdLogs(f.KubeFactory, out),
				//kubecmd.NewCmdAutoscale(f, out),
				NewCmdScale(f.KubeFactory, out),
				kubecmd.NewCmdAttach(f.KubeFactory, in, out, err),
				kubecmd.NewCmdExec(f.KubeFactory, in, out, err),
				rollout.NewCmdRollout(f.KubeFactory, out),
				kubecmd.NewCmdPortForward(f.KubeFactory, out, err),
			},
		},
	}
	groups.Add(cmds)

	filters := []string{"options"}
	templates.ActsAsRootCommand(cmds, filters, groups...)

	if cmds.Flag("namespace") != nil {
		if cmds.Flag("namespace").Annotations == nil {
			cmds.Flag("namespace").Annotations = map[string][]string{}
		}
		cmds.Flag("namespace").Annotations[cobra.BashCompCustom] = append(
			cmds.Flag("namespace").Annotations[cobra.BashCompCustom],
			"__kubectl_get_namespaces",
		)
	}

	cmds.AddCommand(cmdconfig.NewCmdConfig(clientcmd.NewDefaultPathOptions(), out))
	cmds.AddCommand(kubecmd.NewCmdVersion(f.KubeFactory, out))
	cmds.AddCommand(kubecmd.NewCmdOptions(out))

	return cmds
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}
