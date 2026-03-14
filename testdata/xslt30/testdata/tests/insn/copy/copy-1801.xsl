<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" >

  <!-- Purpose: Should be able to copy the same tree fragment twice in succession. -->

<xsl:template match="/">
  <xsl:variable name ="theNode" select="//inc/node4"/>
  <out>
    <xsl:copy-of select="$theNode"/>
    <xsl:copy-of select="$theNode"/>
  </out>
</xsl:template>

</xsl:stylesheet>