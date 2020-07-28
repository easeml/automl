import {ILayoutRestorer, JupyterFrontEnd, JupyterFrontEndPlugin, ILabShell} from '@jupyterlab/application';
import {ICommandPalette, MainAreaWidget, WidgetTracker} from '@jupyterlab/apputils';
import {Message} from '@phosphor/messaging';
import {Widget} from '@phosphor/widgets';
import Vue from 'vue';
import BootstrapVue from 'bootstrap-vue';
import EASEML_VUE from './Iframe.vue'
import EASEML_SIDEBAR_VUE from './Sidebar.vue'
import { ILauncher } from '@jupyterlab/launcher';
import { ISettingRegistry } from '@jupyterlab/coreutils';

const iframeSettings = {easemlServer: ""}

Vue.use(BootstrapVue);

/**
 * Ease.ml main window widget.
 */
class Easeml extends Widget {
    /**
     * Construct a new Ease.ml widget.
     */
    constructor() {
        super();
    }

    /**
     * Handle update requests for the widget.
     */
    async onUpdateRequest(msg: Message): Promise<void> {
        new Vue({
            el: this.node,
            render(h){
                return h(EASEML_VUE, {
                    props: {
                        url: iframeSettings.easemlServer
                    }
                })
            }
        })
    }
}

/**
 * Ease.ml Sidebar widget.
 */
class EasemlSidebar extends Widget {
    /**
     * Construct a new Ease.ml widget.
     */
    constructor() {
        super();
    }

    /**
     * Handle update requests for the widget.
     */
    async onUpdateRequest(msg: Message): Promise<void> {
        new Vue({
            el: this.node,
            render(h){
                return h(EASEML_SIDEBAR_VUE, {
                    props: {

                    }
                })
            }
        })
    }
}

/**
 * Activate the jupyterlab_easml widget extensions and and commands.
 */
function activate(  app: JupyterFrontEnd,
                    palette: ICommandPalette,
                    restorer: ILayoutRestorer,
                    labShell: ILabShell,
                    launcher: ILauncher | null
                     ) {
    // Main Window

    // Open Main window widget
    function easmlOpen(){
        // Declare a widget variable
        let widget: MainAreaWidget<Easeml>;

        if (!widget || widget.isDisposed) {
            // Create a new widget if one does not exist
            const content = new Easeml();
            widget = new MainAreaWidget({content});
            widget.id = 'easeml-jupyterlab';
            widget.title.label = 'Ease.ml';
            widget.title.closable = true;
        }
        if (!tracker.has(widget)) {
            // Track the state of the widget for later restoration
            tracker.add(widget);
        }
        if (!widget.isAttached) {
            // Attach the widget to the main work area if it's not there
            app.shell.add(widget, 'main');
        }
        widget.content.update();

        // Activate the widget
        app.shell.activateById(widget.id);
    }
    // Track and restore the Main widget state
    const tracker = new WidgetTracker<MainAreaWidget<Easeml>>({
        namespace: 'vue'
    });

    // Add an application command that opens the Main widget
    const command: string = 'easeml:open';
    app.commands.addCommand(command, {
        label: 'Open ease.ml',
        execute: easmlOpen
    });

    restorer.restore(tracker, {
        command,
        name: () => 'vue'
    });

    // SIDEBAR

    // Add the command to the palette.
    palette.addItem({command, category: 'START EASE.ML'});

    // Add an application command that closes side widget and opens Main widget
    const sideCommand: string = 'easeml:openside';
    app.commands.addCommand(sideCommand, {
        label: 'Open ease.ml and close sidebar',
        execute: () => {
            easmlOpen();
            labShell.collapseLeft();
        }
    });

    // console.log(app.commands.listCommands())

    // Declare a widget variable
    let sideWidget: MainAreaWidget<EasemlSidebar>;
    // Track and restore the Side Main widget state
    const trackerSide = new WidgetTracker<MainAreaWidget<EasemlSidebar>>({
        namespace: 'vue'
    });

    if (!sideWidget || sideWidget.isDisposed) {
        // Create a new widget if one does not exist
        const content = new EasemlSidebar();
        sideWidget = new MainAreaWidget({content});
        sideWidget.id = 'easeml-jupyterlab';
        sideWidget.title.label = 'Ease.ml';
        sideWidget.title.closable = true;
    }
    if (!trackerSide.has(sideWidget)) {
        // Track the state of the widget for later restoration
        trackerSide.add(sideWidget);
    }
    if (!sideWidget.isAttached) {
        // Attach the widget to the main work area if it's not there
        app.shell.add(sideWidget, 'main');
    }
    sideWidget.content.update();

    // Activate the widget
    app.shell.activateById(sideWidget.id);

    sideWidget.id = "easeml-sidebar";
    sideWidget.title.iconClass = "jp-SpreadsheetIcon jp-SideBar-tabIcon";
    sideWidget.title.caption = "Ease.ml Sidebar";
    sideWidget.title.closable = true;

    restorer.add(sideWidget, sideWidget.id);
    labShell.add(sideWidget, "left");

    // Add a launcher that opens the Main widget area
    if (launcher) {
       launcher.add({
         command: "easeml:open",
         category: 'Other',
         rank: 0
       });
    }
}

/**
 * Reads jupyterlab_easml settings from the Settings Registry
 */
async function loadSettings(app: JupyterFrontEnd, registry: ISettingRegistry){
    try {
        registry.load(plugin.id)
            // .then(easemlSettings => console.log('easemlConfig: ', easemlSettings));

        const reg = await registry.get(plugin.id,"easemlConfig")
            .then(function loadEasemlReg(regVar: any){
                return regVar.composite.easemlServer
            })
        iframeSettings.easemlServer=reg
    } catch (error) {
        // console.error(`Loading ${plugin.id} failed.`, error);
    }
}

/**
 * Initialization data for the jupyterlab_vue extension.
 */
const plugin: JupyterFrontEndPlugin<void> = {
    id: '@easeml/jupyterlab_easeml:plugin',
    requires: [ISettingRegistry,ICommandPalette, ILayoutRestorer,ILabShell],
    optional: [ILauncher],
    activate: async (app: JupyterFrontEnd,
                     registry: ISettingRegistry,
                     palette: ICommandPalette,
                     restorer: ILayoutRestorer,
                     labShell: ILabShell,
                     launcher: ILauncher | null
    ) => {
        await loadSettings(app,registry);
        activate(app,palette,restorer,labShell,launcher);
    },
    autoStart: true
};

const plugins: JupyterFrontEndPlugin<any>[] = [plugin];

export default plugins;
