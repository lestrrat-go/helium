<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that system-property can be dynamically evaluated.-->
   
   <xslt:variable name="expr" select="system-property#1"/>
   <xslt:template match="/">
      <out>
         <xslt:value-of select="$expr('xslt:version') castable as xs:decimal"/>
      </out>
   </xslt:template>
</xslt:transform>
