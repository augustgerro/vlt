# vlt Examples 🚀

Here are some real-world examples of commands you can store in your vault to boost your productivity.

### 🐳 Docker & Containers
| Category | Description | Command |
| :--- | :--- | :--- |
| Docker | Remove all unused data | `docker system prune -af --volumes` |
| Docker | Get IP address of a container | `docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' <container_id>` |
| Docker | Stop and remove all containers | `docker stop $(docker ps -aq) && docker rm $(docker ps -aq)` |

### ☸️ Kubernetes
| Category | Description | Command |
| :--- | :--- | :--- |
| K8s | Delete multiple contexts via fzf | `kubectl config get-contexts -o name \| fzf --multi \| xargs -I {} kubectl config delete-context {}` |
| K8s | Get pods and their nodes | `kubectl get pods -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName` |
| K8s | View secret in plain text | `kubectl get secret <name> -o jsonpath='{.data}' \| jq 'map_values(@base64d)'` |

### 🛠 Git & Development
| Category | Description | Command |
| :--- | :--- | :--- |
| Git | Undo last commit (keep changes) | `git reset --soft HEAD~1` |
| Git | List contributors by commit count | `git shortlog -sn --all` |
| Git | Delete merged local branches | `git branch --merged \| grep -v "\*" \| xargs -n 1 git branch -d` |

### 📂 Filesystem & Search
| Category | Description | Command |
| :--- | :--- | :--- |
| Files | Find large files (>100MB) | `find . -type f -size +100M -exec ls -lh {} +` |
| Files | Replace text in all files recursively | `grep -rl 'old_text' . \| xargs sed -i '' 's/old_text/new_text/g'` |
| Files | Search for text in files via ripgrep | `rg "pattern" --type-add 'web:*.{html,css,js}' --type web` |

### 🍏 MacOS & Network
| Category | Description | Command |
| :--- | :--- | :--- |
| MacOS | Flush DNS cache | `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder` |
| Network | Show active listening ports | `lsof -i -P -n \| grep LISTEN` |
| Network | Get public IP address | `curl -s https://ifconfig.me` |

---

## How to add these to your vault?
Just run:
```bash
vlt "your command here"
```
Or use the manual export to `~/.cli_vault.md`.
