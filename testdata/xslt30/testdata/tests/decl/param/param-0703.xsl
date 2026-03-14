<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:t="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="foo" version="2.0">
    <!-- Purpose: test @visibility is not allowed -->
    
    <!-- Section 9.6 of the spec says: When the static attribute is present with the value yes, 
        the visibility attribute must not have a value other than private.
        
        But in fact, the visibility attribute is not allowed at all: it's prohibited by the grammar -->


    <t:param name="z" visibility="private" static="yes" select="3"/>

    <t:template name="main">
        <out/>
    </t:template>
</t:transform>
