package client

func (c *Client) getConnectedCallback() func() {
	c.muConnectedCallback.RLock()
	defer c.muConnectedCallback.RUnlock()

	return c.connectedCallback
}

func (c *Client) setConnectedCallback(callback func()) {
	c.muConnectedCallback.Lock()
	defer c.muConnectedCallback.Unlock()

	c.connectedCallback = callback
}

func (c *Client) fireConnectedCallback() {
	if callback := c.getConnectedCallback(); callback != nil {
		callback()
	}
}

//=====

func (c *Client) getDisconnectedCallback() func() {
	c.muDisconnectedCallback.RLock()
	defer c.muDisconnectedCallback.RUnlock()

	return c.disconnectedCallback
}

func (c *Client) setDisconnectedCallback(callback func()) {
	c.muDisconnectedCallback.Lock()
	defer c.muDisconnectedCallback.Unlock()

	c.disconnectedCallback = callback
}

func (c *Client) fireDisconnectedCallback() {
	if callback := c.getDisconnectedCallback(); callback != nil {
		callback()
	}
}

//=====

func (c *Client) getPushCallback(route string) (Callback, bool) {
	c.muPushCallbacks.RLock()
	defer c.muPushCallbacks.RUnlock()

	cb, ok := c.pushCallbacks[route]
	return cb, ok
}

func (c *Client) setPushCallback(route string, callback Callback) {
	c.muPushCallbacks.Lock()
	defer c.muPushCallbacks.Unlock()

	c.pushCallbacks[route] = callback
}

func (c *Client) firePushCallback(route string, data []byte) {
	if cb, ok := c.getPushCallback(route); ok && cb != nil {
		cb(data)
	}
}

//=====

func (c *Client) getResponseCallback(mid uint64) (Callback, bool) {
	c.muResponseCallbacks.RLock()
	defer c.muResponseCallbacks.RUnlock()

	cb, ok := c.responseCallbacks[mid]
	return cb, ok
}

func (c *Client) setResponseCallback(mid uint64, cb Callback) {
	c.muResponseCallbacks.Lock()
	defer c.muResponseCallbacks.Unlock()

	if cb == nil {
		delete(c.responseCallbacks, mid)
	} else {
		c.responseCallbacks[mid] = cb
	}
}

func (c *Client) fireResponseCallback(mid uint64, data []byte) {
	if cb, ok := c.getResponseCallback(mid); ok && cb != nil {
		cb(data)
	}
}

//=====

func (c *Client) getKickCallback() Callback {
	c.muKickCallback.RLock()
	defer c.muKickCallback.RUnlock()

	return c.kickCallback
}

func (c *Client) setKickCallback(callback Callback) {
	c.muKickCallback.Lock()
	defer c.muKickCallback.Unlock()

	c.kickCallback = callback
}

func (c *Client) fireKickCallback(data []byte) {
	if callback := c.getKickCallback(); callback != nil {
		callback(data)
	}
}
