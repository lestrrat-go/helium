<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: XSLT 3.0 allows an empty xsl:value-of instruction with no select attribute. -->

   <xslt:template match="/">
      
      <xslt:variable name="x" as="item()*">
        <xslt:value-of/>
      </xslt:variable>
      
   	  <out count="{count($x)}" length="{string-length($x)}" is-text="{$x instance of text()}"/>

	  </xslt:template>
</xslt:transform>
