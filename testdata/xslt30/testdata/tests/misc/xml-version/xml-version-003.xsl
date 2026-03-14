<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: To output a document with method 'xml' and version '1.1'. 
                The name attribute of an xsl:element is a an AVT that has a character reference.  -->

   <t:output method="xml" version="1.1" indent="no"/>

   <t:variable name="v">preØ</t:variable>

   <t:template match="/">
		    <out>
         <t:element name="{$v}" namespace="http://example.org/preÀ"/>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
