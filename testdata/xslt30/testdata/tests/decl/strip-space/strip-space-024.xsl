<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.test1.example.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="foo" version="2.0">
   <!-- Purpose: Test that the "elements" value of xsl:strip-space/xsl:preserve-space can be an empty string. -->

   <t:variable name="var" as="element()">
      <foo:elem>
         <t:value-of select="substring('ab   cd  ef',3,2)"/>
      </foo:elem>
   </t:variable>

   <t:strip-space elements="foo:elem"/>
   <t:strip-space elements=""/>
   <t:strip-space elements="   "/>
   <t:preserve-space elements=""/>
   <t:preserve-space elements="&#x9;"/>

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
