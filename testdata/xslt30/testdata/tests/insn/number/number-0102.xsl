<?xml version="1.0"?> 

<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!--test start-at with @value (xslt 3.0) -->

  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="Family">
    <xsl:for-each select=".//Child">
      <a name="{Name/@First}">
        <xsl:number value="position()" format="1" start-at="0"/>
      </a>
    </xsl:for-each>
  </xsl:template>
  
  <xsl:template match="text()"/>

</xsl:stylesheet>
