<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--
It is a static error if the current-merge-key function is used within a pattern.
-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="x" as="node()"><x/></xsl:variable>
         <xsl:apply-templates select="$x"/>
      </out>
   </xsl:template>
   
   <xsl:template match="x[. = current-merge-key()]"/>
</xsl:stylesheet>
