<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:t="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="foo" version="2.0">
    <!-- Purpose: test @visibility is not allowed -->


    <t:param name="z" visibility="private">
        <t:value-of select="'foo'"/>
    </t:param>

    <t:template name="main">
        <out/>
    </t:template>
</t:transform>
