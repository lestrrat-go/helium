<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#variable-values?>
  <!-- Purpose: Verify top-level xsl:variable, HTML text RTF -->
  <!-- Author: Paul Dick -->

<xsl:variable name="variable"><B>value</B></xsl:variable>
  
<xsl:template match="/">
   <out>
      <xsl:value-of select="$variable"/><xsl:text>,</xsl:text>
      <xsl:copy-of select="$variable"/>
   </out>
</xsl:template>

</xsl:stylesheet>
