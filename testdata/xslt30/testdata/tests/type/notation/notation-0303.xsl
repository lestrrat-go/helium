<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:one="http://notation.example.com"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs one"
                version="2.0">
<!-- Purpose: Test with fn:distinct-values where the argument is a sequence of values of type derived from xs:NOTATION.-->

   <xslt:template match="/one:NOTATION-enumeration">
      <out>
         <xslt:value-of select="distinct-values(one:NOTATION-element/@NOTATION-attribute)"/>
      </out>
   </xslt:template>
</xslt:transform>
