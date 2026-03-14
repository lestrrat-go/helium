<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.test1.example.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="foo" version="2.0">
   <!-- Purpose: Test with an element constructed with whitespace text node children. 
				The element is stored in a variable and its name is listed in xsl:strip-space. 
				Confirm that the element has a text node child. -->

   <t:variable name="var" as="element()">
      <foo:elem>
         <t:value-of select="substring('ab   cd  ef',3,2)"/>
      </foo:elem>
   </t:variable>

   <t:strip-space elements="foo:elem"/>

   <t:template match="/doc">
      <out>
         <a>
            <t:value-of select="$var/text()"/>
         </a>
         <a>
            <t:value-of select="count($var/text())"/>
         </a>
      </out>
   </t:template>
</t:transform>
