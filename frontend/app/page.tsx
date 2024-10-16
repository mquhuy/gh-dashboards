'use client'

import { useState, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@//components/ui/tabs"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@//components/ui/card"
import { Button } from "@//components/ui/button"
import { Switch } from "@//components/ui/switch"
import { ScrollArea } from "@//components/ui/scroll-area"
import { Pin, GitPullRequest, GitPullRequestClosed, GitMerge, CircleDot, ArrowLeft, UserCheck, Mail, MailOpen } from 'lucide-react'
import usePersistentState from './usePersistentState';

interface Notification {
  id: number;
  type: 'PullRequest' | 'Issue';
  repo: string;
  title: string;
  status: 'open' | 'merged' | 'closed';
  pinned: boolean;
  author: string;
  reviewRequested: boolean;
  url: string;
  unread: boolean;
  notifications: { id: number; message: string; timestamp: string; }[];
}

export default function NotificationsManager() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [hideClosedMerged, setHideClosedMerged] = usePersistentState('hideClosedMerged', false)
  const [hideRead, setHideRead] = usePersistentState('hideRead', false)
  const [groupByRepo, setGroupByRepo] = usePersistentState('groupByRepo', false)
  const [selectedItem, setSelectedItem] = useState<Notification | null>(null)
  const [loading, setLoading] = useState(true);
  const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL
  const username = process.env.NEXT_PUBLIC_GITHUB_USERNAME

  const togglePin = (id: number) => {
    setNotifications(notifications.map(n =>
      n.id === id ? { ...n, pinned: !n.pinned } : n
    ))
    const data = {
      action: "togglePin",
      thread_id: id,
    };
    fetch(backendUrl+'/updateThread', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
    });
  }

  const toggleReadStatus = (id: number) => {
    setNotifications(notifications.map(n =>
      n.id === id ? { ...n, unread: !n.unread } : n
    ))
    const data = {
      action: "toggleRead",
      thread_id: id,
    };
    fetch(backendUrl+'/updateThread', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
    });
  }

  const markAsRead = (id: number) => {
    setNotifications(notifications.map(n =>
      n.id === id ? { ...n, unread: false } : n
    ))
    const data = {
      action: "read",
      thread_id: id,
    };
    fetch(backendUrl+'/updateThread', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
    });
  }

  const openURL = (URL: string) => {
    window.open(URL, '_blank', 'noopener,noreferrer');
  }

  const handleItemClick = (item: Notification) => {
    // setSelectedItem(item)
    openURL(item.url)
    markAsRead(item.id)
  }

  const handleBackClick = () => {
    setSelectedItem(null)
  }

  const forcePull = async () => {
    try {
      const response = await fetch(backendUrl + '/forcePull', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      if (!response.ok) {
        throw new Error('Failed to force pull notifications');
      }
      fetchNotifications();
    } catch (error) {
      console.error('Error:', error);
    }
  };

  const fetchNotifications = async () => {
    const response = await fetch(backendUrl+'/threads'); // Adjust this URL if needed
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    const data = await response.json();
    setNotifications(data)
    setLoading(false); // Set loading to false after fetching data
  };

  useEffect(() => {
    fetchNotifications();
    const intervalId = setInterval(fetchNotifications, 300000); // Fetch notifications every 5 minutes
    return () => clearInterval(intervalId); // Cleanup interval on component unmount
  }, []);

  const unmergedNotifications = notifications.filter(n =>
    !hideClosedMerged || (n.status !== 'merged' && n.status !== 'closed')
  )

  const filteredNotifications = unmergedNotifications.filter(n =>
    !hideRead || (n.unread)
  )

  // Group notifications by repo if groupByRepo is true
  const groupedNotifications = groupByRepo
    ? notifications.reduce((acc, notification) => {
        if (hideRead && !notification.unread) {
          console.log('Skipping read notification');
          return acc;
        }
        if (hideClosedMerged && (notification.status === 'merged' || notification.status === 'closed')) {
          return acc;
        }
        const repo = notification.repo;
        if (!acc[repo]) {
          acc[repo] = [];
        }
        acc[repo].push(notification);
        return acc;
      }, {} as { [key: string]: Notification[] })
    : { 'All Repositories': filteredNotifications };

  const pinnedNotifications = notifications.filter(n => n.pinned)
  const myPRs = notifications.filter(n => n.type === 'PullRequest' && n.author === username)
  const myIssues = notifications.filter(n => n.type === 'Issue' && n.author === username)
  const reviewRequested = notifications.filter(n => n.reviewRequested)

  if (loading) return <p>Loading...</p>

  return (
    <Card className="w-full max-w-4xl mx-auto">
      <CardHeader>
        <CardTitle>GitHub Notifications Manager</CardTitle>
        <CardDescription>Manage your GitHub notifications, PRs, and Issues</CardDescription>
      </CardHeader>
      <CardContent>
        {selectedItem ? (
          <div>
            <Button variant="ghost" onClick={handleBackClick} className="mb-4">
              <ArrowLeft className="mr-2 h-4 w-4" /> Back to list
            </Button>
            <h2 className="text-xl font-bold mb-2">{selectedItem.title}</h2>
            <p className="text-sm text-gray-500 mb-4">
              {selectedItem.type.toUpperCase()} in {selectedItem.repo} • {selectedItem.status}
            </p>
            <h3 className="text-lg font-semibold mb-2">Notifications</h3>
            {selectedItem.notifications.length > 0 ? (
              <ul className="space-y-2">
                {selectedItem.notifications.map(notification => (
                  <li key={notification.id} className="bg-gray-100 p-2 rounded">
                    <p>{notification.message}</p>
                    <p className="text-xs text-gray-500">{new Date(notification.timestamp).toLocaleString()}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <p>No notifications for this item.</p>
            )}
          </div>
        ) : (
          <>
            <div className="flex justify-between items-center mb-4">
              <Button variant="primary" onClick={forcePull}>
                Force Pull Notifications
              </Button>
              <div className="flex items-center space-x-2">
                <Switch
                  id="hide-closed-merged"
                  checked={hideClosedMerged}
                  onCheckedChange={setHideClosedMerged}
                />
                <label htmlFor="hide-closed-merged">Hide Closed/Merged</label>
              </div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="hide-read"
                  checked={hideRead}
                  onCheckedChange={setHideRead}
                />
                <label htmlFor="hide-read">Hide Read</label>
              </div>
              <div className="flex items-center space-x-2">
                <Switch
                  id="group-by-repo"
                  checked={groupByRepo}
                  onCheckedChange={setGroupByRepo}
                />
                <label htmlFor="group-by-repo">Group by Repository</label>
              </div>
            </div>

            <Tabs defaultValue="all" className="w-full">
              <TabsList>
                <TabsTrigger value="all">All</TabsTrigger>
                <TabsTrigger value="prs">Pull Requests</TabsTrigger>
                <TabsTrigger value="issues">Issues</TabsTrigger>
                <TabsTrigger value="my-prs">PRs by me</TabsTrigger>
                <TabsTrigger value="my-issues">Issues by me</TabsTrigger>
                <TabsTrigger value="review-requested">Review Requested</TabsTrigger>
                <TabsTrigger value="pinned">Pinned</TabsTrigger>
              </TabsList>

              <TabsContent value="all">
                <NotificationList
                  notifications={groupedNotifications}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={!groupByRepo}
                  filterType=""
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="prs">
                <NotificationList
                  notifications={groupedNotifications}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={!groupByRepo}
                  filterType="PullRequest"
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="issues">
                <NotificationList
                  notifications={groupedNotifications}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={!groupByRepo}
                  filterType="Issue"
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="my-prs">
                <NotificationList
                  notifications={{ 'My Pull Requests': myPRs }}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={true}
                  filterType="PullRequest"
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="my-issues">
                <NotificationList
                  notifications={{ 'My Issues': myIssues }}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={true}
                  filterType="Issue"
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="review-requested">
                <NotificationList
                  notifications={{ 'Review Requested': reviewRequested }}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={true}
                  filterType="PullRequest"
                  onItemClick={handleItemClick}
                />
              </TabsContent>

              <TabsContent value="pinned">
                <NotificationList
                  notifications={{ 'Pinned Items': pinnedNotifications }}
                  togglePin={togglePin}
                  toggleReadStatus={toggleReadStatus}
                  showRepo={true}
                  filterType=""
                  onItemClick={handleItemClick}
                />
              </TabsContent>
            </Tabs>
          </>
        )}
      </CardContent>
    </Card>
  )
}

function NotificationList({ notifications, togglePin, toggleReadStatus, showRepo, filterType, onItemClick }) {
  return (
    <ScrollArea className="flex-1 overflow-y-auto p-4">
      {Object.entries(notifications).map(([repo, items]) => (
        <div key={repo}>
          <h3 className="font-semibold mb-2">{repo}</h3>
          {items
            .filter(item => !filterType || item.type === filterType)
            .map(item => (
              <div key={item.id} className="flex items-center justify-between py-2 border-b last:border-b-0">
                <div
                  className={`flex items-center space-x-2 cursor-pointer ${item.unread ? 'font-bold' : ''}`}
                  onClick={() => onItemClick(item)}
                >
                  {item.type === 'PullRequest' ? (
                    <GitPullRequest className={`w-5 h-5 ${item.unread ? 'text-green-600' : 'text-green-500'}`} />
                  ) : (
                    <CircleDot className={`w-5 h-5 ${item.unread ? 'text-green-600' : 'text-green-500'}`} />
                  )}
                  <span className={`${item.unread ? 'text-gray-900' : 'text-gray-700'}`}>{item.title}</span>
                  {showRepo && <span className={`text-sm ${item.unread ? 'text-gray-700' : 'text-gray-500'}`}>({item.repo})</span>}
                  {item.status == 'merged' && <GitMerge className="w-4 h-4 text-purple-500" />}
                  {item.status == 'closed' && <GitPullRequestClosed className="w-4 h-4 text-red-500" />}
                  {item.reviewRequested && <UserCheck className="w-4 h-4 text-blue-500 ml-2" />}
                </div>
                <div className="flex items-center space-x-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleReadStatus(item.id);
                    }}
                    title={item.unread ? "Mark as read" : "Mark as unread"}
                  >
                    {item.unread ? (
                      <Mail className="w-4 h-4 text-blue-500" />
                    ) : (
                      <MailOpen className="w-4 h-4 text-gray-500" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={(e) => {
                      e.stopPropagation();
                      togglePin(item.id);
                    }}
                    title={item.pinned ? "Unpin" : "Pin"}
                  >
                    <Pin className={`w-4 h-4 ${item.pinned ? 'text-yellow-500 fill-yellow-500' : 'text-gray-500'}`} />
                  </Button>
                </div>
              </div>
            ))}
        </div>
      ))}
    </ScrollArea>
  )
}
