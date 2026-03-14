<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output method="xml"/>

<?spec xslt#built-in-rule?>
    <!-- Purpose: Test of built-in template for text nodes. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="following-sibling::node()"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="node()">
   <xsl:value-of select="position()"/>node:<xsl:value-of select="name()"/>,
</xsl:template>

<!-- For testing this should remain commented out. It shows how all nodes
     are processed, particularly the text nodes.

<xsl:template match="text()">
   <xsl:value-of select="position()"/>text:<xsl:value-of select="."/>,
</xsl:template>

-->

</xsl:stylesheet>