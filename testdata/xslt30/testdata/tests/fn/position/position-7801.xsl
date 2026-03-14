<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position93 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() in match pattern. Nodes are sorted. -->
  <!-- Elaboration: The nodes are processed in the sorted order, but the position() in the
     match pattern always refers to the node as found in the source tree. -->

<xsl:template match="doc">
  <out><xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="a">
      <xsl:sort select="@z" data-type="number"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="a[position()=4]">
  <a4><xsl:value-of select="."/></a4><xsl:text>&#10;</xsl:text>
</xsl:template>

<xsl:template match="a[position()=3]">
  <a3><xsl:value-of select="."/></a3><xsl:text>&#10;</xsl:text>
</xsl:template>

<xsl:template match="a[position()=2]">
  <a2><xsl:value-of select="."/></a2><xsl:text>&#10;</xsl:text>
</xsl:template>

<xsl:template match="a[position()=1]">
  <a1><xsl:value-of select="."/></a1><xsl:text>&#10;</xsl:text>
</xsl:template>

</xsl:stylesheet>