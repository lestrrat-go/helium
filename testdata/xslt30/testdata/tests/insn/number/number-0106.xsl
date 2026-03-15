<?xml version="1.0"?> 

<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!--test start-at with @value (xslt 3.0) - level="multiple" -->

  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="Family">
    <xsl:for-each select=".//Child">
      <a name="{Name/@First}">
        <xsl:number select="." level="multiple" count="*" format="a.a.a" start-at="3 2 1"/>
      </a>
    </xsl:for-each>
  </xsl:template>

  <xsl:template match="text()"/>

</xsl:stylesheet>
