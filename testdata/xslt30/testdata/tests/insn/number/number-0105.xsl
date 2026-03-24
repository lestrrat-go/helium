<?xml version="1.0"?> 

<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!--test start-at with @value (xslt 3.0) - negative start -->

  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:param name="start" select="-6"/>

  <xsl:template match="Family">
    <xsl:for-each select=".//Child">
      <a name="{Name/@First}">
        <xsl:number select="." level="any" count="*" format="Ww" start-at="{$start}" lang="en"/>
      </a>
    </xsl:for-each>
  </xsl:template>
  
  <xsl:template match="text()"/>

</xsl:stylesheet>
