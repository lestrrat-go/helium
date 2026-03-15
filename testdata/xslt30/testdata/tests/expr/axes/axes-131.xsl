<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
<!-- Purpose: Test for descendant axis when siblings have some children or
     the context node has no siblings. -->

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="doc">
  <out>
    <xsl:for-each select="child">
      <xsl:text>&#10;</xsl:text>
      <child number="{position()}">
        <xsl:for-each select="grandchild">
          <xsl:text>&#10;</xsl:text>
          <grandchild number="{position()}">
            <xsl:attribute name="descendantaxiscount"><xsl:value-of select="count(descendant::*)"/></xsl:attribute>
            <xsl:attribute name="descendantorselfaxiscount"><xsl:value-of select="count(descendant-or-self::*)"/></xsl:attribute>
          </grandchild>
        </xsl:for-each>
        <xsl:text>&#10;</xsl:text>
      </child>
    </xsl:for-each>
    <xsl:text>&#10;</xsl:text>
  </out>
</xsl:template>

</xsl:stylesheet>