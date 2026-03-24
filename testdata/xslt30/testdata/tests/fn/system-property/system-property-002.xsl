<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

<!-- Purpose: Test that xsl:vendor is a valid system-property and the returned result is an xs:string.-->

   <xsl:template match="/">
      <out>
         <xsl:value-of select="system-property('xsl:vendor') instance of xs:string"/>
      </out>
   </xsl:template>
</xsl:transform>
