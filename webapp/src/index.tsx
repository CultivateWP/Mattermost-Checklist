// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from 'manifest';

import ChecklistPost from 'components/checklist_post';

import type {PluginRegistry} from 'types/mattermost-webapp';

export default class Plugin {
    public async initialize(registry: PluginRegistry) {
        registry.registerPostTypeComponent(checklistPostType, ChecklistPost);
    }
}

const checklistPostType = 'custom_checklist';

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
