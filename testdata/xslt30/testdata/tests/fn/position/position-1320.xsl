<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position51 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() when used with key() in xsl:apply-templates. Look for first. -->

<xsl:key name="k" match="a" use="@test"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="key('k','true')[1]/num"/>
  </out>
</xsl:template>

<xsl:template match="num">
  <B><xsl:value-of select="."/></B>
</xsl:template>

</xsl:stylesheet>
