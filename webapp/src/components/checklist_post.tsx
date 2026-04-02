import manifest from 'manifest';
import React, {useEffect, useMemo, useState} from 'react';

import type {Post} from '@mattermost/types/posts';

import './checklist_post.scss';

type ChecklistItem = {
    id: string;
    text: string;
    checked: boolean;
    checked_at?: number;
    checked_by?: string;
    checked_by_username?: string;
};

type Checklist = {
    title: string;
    creator_id?: string;
    updated_at?: number;
    items: ChecklistItem[];
};

type ToggleResponse = {
    checklist?: Checklist;
    error?: string;
};

type Props = {
    post: Post;
};

type WindowWithBasename = Window & {
    basename?: string;
};

const checklistPropsKey = 'checklist';

function getPluginURL(path: string): string {
    const basename = (window as WindowWithBasename).basename || '';
    return `${basename}/plugins/${manifest.id}${path}`;
}

function readChecklist(post: Post): Checklist | null {
    const rawChecklist = post.props?.[checklistPropsKey];
    if (!rawChecklist || typeof rawChecklist !== 'object') {
        return null;
    }

    return rawChecklist as Checklist;
}

function completionSummary(checklist: Checklist): string {
    const completed = checklist.items.filter((item) => item.checked).length;
    return `${completed}/${checklist.items.length} done`;
}

export default function ChecklistPost({post}: Props) {
    const [checklist, setChecklist] = useState<Checklist | null>(() => readChecklist(post));
    const [pendingItemId, setPendingItemId] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        setChecklist(readChecklist(post));
    }, [post]);

    const summary = useMemo(() => {
        if (!checklist) {
            return '';
        }

        return completionSummary(checklist);
    }, [checklist]);

    const toggleItem = async (itemId: string) => {
        if (!checklist || pendingItemId) {
            return;
        }

        setPendingItemId(itemId);
        setError(null);

        try {
            const response = await fetch(
                getPluginURL(`/api/v1/checklists/${post.id}/items/${itemId}/toggle`),
                {
                    method: 'POST',
                    credentials: 'same-origin',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                },
            );

            const data = await response.json() as ToggleResponse;
            if (!response.ok || !data.checklist) {
                throw new Error(data.error || 'Unable to update this checklist right now.');
            }

            setChecklist(data.checklist);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unable to update this checklist right now.');
        } finally {
            setPendingItemId(null);
        }
    };

    if (!checklist) {
        return null;
    }

    return (
        <div className='mm-checklist-post'>
            <div className='mm-checklist-post__header'>
                <div>
                    <span className='mm-checklist-post__eyebrow'>{'Interactive checklist'}</span>
                    <h4 className='mm-checklist-post__title'>{checklist.title || 'Checklist'}</h4>
                </div>
                <span className='mm-checklist-post__summary'>{summary}</span>
            </div>
            <div className='mm-checklist-post__items'>
                {checklist.items.map((item) => {
                    const isPending = pendingItemId === item.id;
                    const meta = item.checked ?
                        `Checked by ${item.checked_by_username ? '@' + item.checked_by_username : 'a teammate'}` :
                        'Open';

                    return (
                        <button
                            type='button'
                            key={item.id}
                            className={`mm-checklist-post__item ${item.checked ? 'mm-checklist-post__item--checked' : ''}`}
                            onClick={() => toggleItem(item.id)}
                            disabled={Boolean(pendingItemId)}
                        >
                            <span className='mm-checklist-post__checkbox'>{item.checked ? '✓' : ''}</span>
                            <span className='mm-checklist-post__body'>
                                <span className='mm-checklist-post__item-text'>{item.text}</span>
                                <span className='mm-checklist-post__meta'>{meta}</span>
                            </span>
                            <span className='mm-checklist-post__status'>{isPending ? 'Saving...' : ''}</span>
                        </button>
                    );
                })}
            </div>
            {error ? <div className='mm-checklist-post__error'>{error}</div> : null}
        </div>
    );
}
