<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Check that namespace axis includes all namespaces in scope. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:element name="{name(.)}">
     <xsl:for-each select="namespace::*">
		<xsl:sort select="name(.)"/>
        <xsl:element name="{name(.)}"><xsl:value-of select="."/></xsl:element>
     </xsl:for-each>
  </xsl:element>
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>
