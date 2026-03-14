<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Define global variable using a when with a dependency on a variable in an included file -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:include href="variable-4701a.xsl"/>

<xsl:variable name="bar">
  <xsl:choose>
    <xsl:when test="$foo='bar'">
      <xsl:text>the value of bar is bar</xsl:text>
    </xsl:when>
    <xsl:otherwise>
      <xsl:text>the value of bar is undefined</xsl:text>
    </xsl:otherwise>
  </xsl:choose>
</xsl:variable>

<xsl:template match="/">
  <out>
    <xsl:value-of select="$bar"/>
  </out>
</xsl:template>

</xsl:stylesheet>