<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position41 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test of position() in a for-each node set involving parent axis. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="a/x"/>
  </out>
</xsl:template>

<xsl:template match="x">
  <xsl:text>From the x node...
</xsl:text>
  <xsl:for-each select="../../a/b[1]">
    <xsl:if test="position() &gt;= 2 and position() &lt;= 6">
      <xsl:value-of select="."/>
      <xsl:text>
</xsl:text>
    </xsl:if>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
