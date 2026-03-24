<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:last()'s return value as xs:integer -->

   <xslt:template match="doc">
      <out>
         <xslt:value-of select="last()"/>
         <xslt:text> | </xslt:text>
         <xslt:value-of select="last() instance of xs:integer"/>
      </out>
   </xslt:template>
</xslt:transform>
