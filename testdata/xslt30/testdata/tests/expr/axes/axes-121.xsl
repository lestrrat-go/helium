<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Section: 2.2 -->
  <!-- Purpose: Test for '/descendant::*' to select all elements (excluding root node).
    No text or comments should be picked up. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:for-each select="/descendant::*">
      <xsl:value-of select="name(.)"/>
      <xsl:choose>
        <xsl:when test="name(.)='center'">
          <xsl:text>&#10;</xsl:text>
        </xsl:when>
        <xsl:otherwise>
          <xsl:text> </xsl:text>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
