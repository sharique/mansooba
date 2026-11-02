# Roles and permissions

Who can do what in Mansooba. There are two separate things:

- **Project roles** (admin, member, viewer) decide what a person can do **inside one project**.
- The **system administrator** setting decides who looks after **the installation** (accounts, settings, logs). It gives no access to the inside of any project.

## Project roles

| Role | In one line |
|---|---|
| **Admin** | Looks after the project and its people. Can do everything a member can, plus manage sprints, people and the project itself. |
| **Member** | Does the work: tasks, comments, labels and files. |
| **Viewer** | Can read everything in the project and change nothing. |

A person has exactly one role in each project they belong to. A person who is not in a project cannot see it, and is told "not found", the same as for a project that does not exist.

### What each role can do

<!-- This table is checked against the rules in the code by a test (backend/internal/domain/access_doc_test.go).
     Keep the capability names, the column order and the yes/blank marks exactly as they are. -->

| Capability | Viewer | Member | Admin |
|---|---|---|---|
| `project.view` | yes | yes | yes |
| `task.write` | | yes | yes |
| `task.delete_any` | | | yes |
| `comment.write` | | yes | yes |
| `comment.delete_any` | | | yes |
| `label.write` | | yes | yes |
| `label.delete` | | | yes |
| `attachment.write` | | yes | yes |
| `attachment.delete_any` | | | yes |
| `sprint.manage` | | | yes |
| `project.edit` | | | yes |
| `member.manage` | | | yes |
| `project.delete` | | | yes |

In plain words:

| Capability | Means |
|---|---|
| `project.view` | Read the project: tasks, comments, sprints, the board, the backlog, reports, members |
| `task.write` | Create and edit tasks, change status, assign, link tasks, move cards on the board; delete **your own** tasks |
| `task.delete_any` | Delete other people's tasks |
| `comment.write` | Comment; edit and delete **your own** comments |
| `comment.delete_any` | Delete other people's comments |
| `label.write` | Create labels; add them to and remove them from tasks |
| `label.delete` | Remove a label from the project |
| `attachment.write` | Attach files; delete **your own** files |
| `attachment.delete_any` | Delete other people's files |
| `sprint.manage` | Create, edit, start, complete and delete sprints |
| `project.edit` | Change the project's name and description |
| `member.manage` | Add people, remove people, change someone's role |
| `project.delete` | Delete the project (it must have no tasks left, and the name is typed to confirm) |

"Your own" means you made it **and** your role can still make such things. A member who is later changed to viewer can no longer edit or delete what they wrote.

### Rules that always hold

- **A project always has at least one active admin.** Demoting or removing the last one is refused. Admins cannot remove themselves; they can step down by changing their own role while another admin remains.
- **Role changes apply on the person's next action.** No new sign-in is needed.
- **Anyone who can sign in can create a project** and is its admin from the start. Being the creator gives no right beyond that: another admin can change the creator's role like anyone else's.
- **A task can only be assigned to a member of its project** (any role, a viewer included). Assigning needs permission to edit tasks.
- **Leaving a project takes its tasks with you.** Someone removed from a project no longer sees its tasks or activity in their own "my tasks" and "my activity" lists.
- **Mentions reach members only.** Mentioning someone outside the project sends no notification, because a notification names the task and the project.
- **Other projects' ids are not reachable.** A sprint, label or task from another project, used inside this project's URL or request, is treated as not found.
- Changing roles, adding and removing people, and deleting projects are written to the **System Logs**, and the person affected is notified.

## The system administrator

A setting on a person's account. The first account created by the setup wizard has it; afterwards only another system administrator can give it or take it away (System → Users).

It covers:

- **Users and accounts:** list, create, enable, disable, make or remove system administrators
- **System settings:** change them
- **System logs:** view them

It does **not** open any project. A system administrator who is not in a project cannot see its tasks, board or members. To work in a project, a project admin adds them like anyone else, with an ordinary role.

### Always at least one

An installation always has at least one **active** system administrator. Demoting or disabling the last one is refused, even if two administrators try at the same moment.

### Projects with no admin

If every admin of a project has had their account disabled, the project has no one to look after it. A system administrator can then:

1. open **System → Projects without an admin**, which lists such projects by key and name only;
2. choose **Take admin** on one. This makes them its admin (and only for a project that has none), writes it to the System Logs and notifies the project's members. Until they do this they still cannot see inside the project.

### If no one can sign in as an administrator

If administrator accounts exist but all are disabled, run the recovery command on the server:

```bash
./restore-admin --email you@example.com
```

This enables that account and makes it a system administrator. (The setup wizard is not reopened for this case, so a stranger cannot take over an installation.)
