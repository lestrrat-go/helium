<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:foo="http://www.ped.com">

<?spec xpath#node-tests?>
    <!-- Purpose: Check the attribute:: axis. The foo:doc selection is necessary
       to pick up the last doc, cuz, I change the default namespace. -->

<xsl:template match="docs">
 <out>
    <xsl:for-each select="doc">
      <xsl:apply-templates select="attribute::*">
        <xsl:sort select="local-name()"/>
      </xsl:apply-templates>
    </xsl:for-each>
    <xsl:for-each select="foo:doc">
      <xsl:apply-templates select="attribute::*">
        <xsl:sort select="local-name()"/>
      </xsl:apply-templates>
    </xsl:for-each>
 </out>
</xsl:template>

<xsl:template match="@*">
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>