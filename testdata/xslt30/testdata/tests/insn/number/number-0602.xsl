<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE test group separation in number formatting -->
  <xsl:template match="doc">
    <out>
      <xsl:number value="1000000" grouping-separator="/" grouping-size="2"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
